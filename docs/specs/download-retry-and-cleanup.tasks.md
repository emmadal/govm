# Plan : téléchargement fiable (retry, timeout, nettoyage)

Spec : [download-retry-and-cleanup.md](download-retry-and-cleanup.md) · Statut : approuvé et implémenté (T1–T9) le 2026-11-12, non commité

Ordre : le téléchargeur (risque le plus élevé) d'abord, isolé et testé sans toucher au CLI ; ensuite branchement commande par commande ; correctifs indépendants à la fin. Chaque tâche = un commit, `go build ./...` vert après chacune. Méthode : test en échec d'abord, puis le code.

## Tâches

### T1 — Tentative unique avec timeout d'inactivité
- **But** : une fonction télécharge une URL vers `<dest>.part` et échoue si aucun octet n'arrive pendant le délai d'inactivité.
- **Fichiers** : `pkg/download.go` (nouveau), `pkg/download_test.go` (nouveau)
- **Détail** : `context.WithCancel` + minuteur réarmé à chaque `Read` ; le délai couvre aussi l'attente des en-têtes. Durées passées par une struct d'options (60 s par défaut) pour que les tests tournent en millisecondes. Aucun `http.Client.Timeout` global.
- **Vérification** : `go test ./pkg -run 'TestDownload_Idle' -race` — serveur lent mais actif : succès ; serveur silencieux : erreur de timeout ; constante par défaut = 60 s.
- **Couvre** : AC4 · **Dépend de** : — · **Risque** : élevé (cœur du correctif, concurrence minuteur/lecture)

### T2 — Écriture atomique et nettoyage du fichier partiel
- **But** : le fichier final n'existe que si le téléchargement est complet ; tout échec supprime le `.part`.
- **Fichiers** : `pkg/download.go`, `pkg/download_test.go`
- **Détail** : `.part` au nom unique par processus (`os.CreateTemp` dans le dossier de destination), contrôle taille reçue = `Content-Length` quand il est connu, `Close` avant `Rename`/`Remove` (Windows), erreur de `Close` remontée.
- **Vérification** : `go test ./pkg -run 'TestDownload_(Atomic|NoContentLength|Truncated|ReplacesLarger)' -race`
- **Couvre** : AC6, AC7 (côté téléchargeur), AC8 · **Dépend de** : T1 · **Risque** : moyen (sémantique de `rename` sous Windows)

### T3 — Boucle de 3 tentatives et classement des erreurs
- **But** : les erreurs transitoires sont retentées jusqu'à 3 tentatives au total, les erreurs définitives échouent tout de suite.
- **Fichiers** : `pkg/download.go`, `pkg/download_test.go`
- **Détail** : retenté = timeout, coupure, EOF prématuré, taille incorrecte, 5xx, 429 ; non retenté = autres 4xx, annulation, erreur disque. Pause 1 s puis 2 s (injectable). Erreur typée pour le statut HTTP ; erreur finale `… after 3 attempts: <cause>`. Callback `OnRetry(n, max)` pour l'affichage.
- **Vérification** : `go test ./pkg -run 'TestDownload_Retry' -race` — compteur de requêtes = 3 (2 blocages puis succès), = 3 (échec permanent, dossier vide ensuite), = 1 (404), = 3 (500).
- **Couvre** : AC1, AC2, AC3, AC5 (côté téléchargeur) · **Dépend de** : T2 · **Risque** : moyen

### T4 — Annulation par Ctrl-C
- **But** : Ctrl-C arrête le téléchargement sans nouvelle tentative et sans laisser de `.part`.
- **Fichiers** : `main.go`, `cmd/root.go`, `pkg/download_test.go`
- **Détail** : `signal.NotifyContext` dans `main`, `initCmd.ExecuteContext(ctx)` ; le téléchargeur reçoit `cmd.Context()`.
- **Vérification** : `go test ./pkg -run 'TestDownload_Cancel' -race` (contexte annulé en cours de lecture : 1 requête, dossier vide) + essai manuel `go run . install 1.25.0` puis Ctrl-C, `ls ~/.govm/.cache`.
- **Couvre** : AC11 · **Dépend de** : T3 · **Risque** : faible

**Jalon A** — `go test ./pkg -race` vert, CLI inchangé. Point de contrôle avant de brancher.

### T5 — `govm install` utilise le téléchargeur
- **But** : `DownloadGoVersion` délègue au téléchargeur et retourne le chemin de l'archive.
- **Fichiers** : `pkg/manager.go`, `cmd/install.go`, `pkg/manager_test.go` (nouveau)
- **Détail** : suppression du champ `Tarball.File` (source du panic B3) au profit d'un chemin ; URL de base surchargeable pour les tests ; barre de progression recréée à chaque tentative (barre indéterminée si taille inconnue) ; message `Retrying (2/3)…` ; 404 → « no version go<X> found », autres statuts → message avec le code HTTP ; correction du `%s` sur `*os.File` (B5).
- **Vérification** : `go test ./pkg -run 'TestDownloadGoVersion' -race` + essai manuel `go run . install 1.25.0`
- **Couvre** : AC5, AC7, B1–B5 · **Dépend de** : T4 · **Risque** : moyen (seul chemin d'installation existant)

### T6 — Nettoyage après extraction en échec
- **But** : une extraction ratée supprime le dossier `go<version>` et l'archive du cache.
- **Fichiers** : `pkg/manager.go`, `pkg/manager_test.go`
- **Vérification** : `go test ./pkg -run 'TestInstallVersion_CorruptArchive'` — archive invalide : erreur, dossier de version absent, archive absente.
- **Couvre** : AC10 (B8) · **Dépend de** : T5 · **Risque** : faible

### T7 — Validation des arguments sans panic
- **But** : `install` et `use` rejettent une entrée absente ou mal formée avec une erreur lisible.
- **Fichiers** : `cmd/install.go`, `cmd/use.go`, `cmd/version.go` (nouveau, validation partagée), `cmd/version_test.go` (nouveau)
- **Détail** : contrôle de `len(args)` avant `args[0]` ; version validée par motif `N.N.N` avant `compareVersions`.
- **Vérification** : `go test ./cmd` + `go run . install; go run . use; go run . install abcdef` — code de sortie 1, aucune trace de panic.
- **Couvre** : AC9 (B6, B7) · **Dépend de** : — · **Risque** : faible

### T8 — `govm update` utilise le téléchargeur
- **But** : le binaire `govm` est téléchargé avec les mêmes 3 tentatives, timeout d'inactivité et nettoyage.
- **Fichiers** : `internal/update.go`, `cmd/update.go`, `internal/update_test.go` (nouveau)
- **Détail** : remplacement de `http.Get` ; URL de base surchargeable pour les tests ; `copyFile` avec `O_TRUNC`.
- **Vérification** : `go test ./internal -run 'TestDownloadLatestRelease' -race`
- **Couvre** : AC12 (B9) · **Dépend de** : T4 · **Risque** : moyen (un échec ici casse la mise à jour automatique ; repli = `scripts/install.sh`)

### T9 — `go vet` propre et tests en CI
- **But** : `go vet` et `go test` passent, et la CI les exécute sur chaque pull request.
- **Fichiers** : `internal/uninstall.go` (ligne 208, chaîne de format non constante), `.github/workflows/linter.yml`
- **Détail** : étape `go vet ./... && go test ./... -race` sur ubuntu, macos et windows (le comportement de `rename` diffère sous Windows).
- **Vérification** : `go vet ./... && go test ./... -race`
- **Couvre** : AC13 · **Dépend de** : T1–T8 · **Risque** : faible

**Jalon B** — les 13 critères passent ; essai manuel de bout en bout : `install`, `use`, `list`, `rm`, `update`.

## Couverture des critères

| Critère | AC1 | AC2 | AC3 | AC4 | AC5 | AC6 | AC7 | AC8 | AC9 | AC10 | AC11 | AC12 | AC13 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Tâche | T3 | T3 | T3 | T1 | T3, T5 | T2 | T2, T5 | T2 | T7 | T6 | T4 | T8 | T9 |

## Déploiement et retour arrière

- **Livraison** : `VERSION` 1.0.1 → 1.0.2, tag `v1.0.2`, workflow `release.yml` existant. Pas de flag.
- **Retour arrière** : supprimer la release `v1.0.2` pour que `releases/latest` repointe sur 1.0.1 ; les utilisateurs déjà mis à jour réinstallent par `scripts/install.sh`. Aucun format de données modifié.
- **Hors plan** : commit, tag et push se font sur demande explicite.

## Risques ouverts

- `release.yml` compile avec Go 1.24 alors que `go.mod` exige 1.26.4 : à vérifier avant le tag (hors périmètre de la spec).
- Sous Windows, `govm update` écrase un exécutable en cours d'exécution : comportement actuel conservé, non traité ici.
