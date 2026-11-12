# Spec : téléchargement fiable (retry, timeout, nettoyage)

Statut : validée — 2026-11-12

## 1. Problème

`govm install <version>` télécharge l'archive Go avec `http.Client{Timeout: 60s}` (`pkg/manager.go:57`). Ce timeout couvre **tout** le téléchargement, corps compris : sur une connexion lente, une archive de ~70 Mo est coupée à 60 s. Il n'y a aucune nouvelle tentative, et le fichier partiel reste dans `~/.govm/.cache`. `govm use` et `govm rm` considèrent ensuite ce fichier comme une version valide (`pkg/pkg.go:36`), et une réinstallation réécrit par-dessus sans tronquer.

Bugs voisins constatés à la lecture du code :

| # | Emplacement | Défaut |
|---|---|---|
| B1 | `pkg/manager.go:57` | Timeout global de 60 s, pas de retry |
| B2 | `pkg/manager.go:72` | Écriture directe dans le fichier final, sans `O_TRUNC`, jamais supprimé en cas d'échec |
| B3 | `pkg/manager.go:84` + `cmd/install.go:57` | `Content-Length` absent : rien n'est écrit, `tarball.File` reste `nil`, panic |
| B4 | `pkg/manager.go:65` | Tout statut non-200 (500, 429, 403) affiché comme « no version found » |
| B5 | `pkg/manager.go:74,76` | `%s` appliqué à un `*os.File` nil (`go vet`) ; erreur de `Close` perdue (retour non nommé) |
| B6 | `cmd/install.go:19`, `cmd/use.go:19` | `args[0]` lu avant le contrôle de `len(args)` : panic sans argument |
| B7 | `cmd/install.go:74` | `compareVersions` panic si la version n'a pas 3 segments (ex. `abcdef`) |
| B8 | `pkg/manager.go:96` | Échec d'extraction : dossier `go<version>` partiel conservé, vu comme installé |
| B9 | `internal/update.go:66,117` | `govm update` : `http.Get` sans timeout, pas de retry, copie sans `O_TRUNC` |

## 2. Objectifs / non-objectifs

**Objectifs**
- Un téléchargement interrompu par timeout est retenté, 3 tentatives au total.
- Une tentative n'est déclarée en timeout qu'après 60 s sans aucun octet reçu ; pas de limite sur la durée totale.
- Après échec définitif, aucun fichier partiel ne subsiste et la commande sort en erreur.
- Corriger B2 à B8 ; appliquer la même logique de téléchargement à `govm update` (B9).

**Non-objectifs**
- Reprise partielle (`Range`), téléchargements parallèles, miroirs.
- Vérification SHA-256 de l'archive (spec séparée).
- Flags de configuration du nombre de tentatives ou du timeout.
- Refonte de l'extraction (`tar` via shell) ou de la gestion du `PATH`.

## 3. Critères d'acceptation

Tests avec `httptest.Server`, durées injectables (timeout et pause ramenés à quelques millisecondes).

1. **AC1** — Serveur qui bloque 2 fois puis répond : l'installation réussit, le serveur a reçu exactement 3 requêtes.
2. **AC2** — Serveur qui bloque toujours : exactement 3 requêtes, erreur retournée contenant `after 3 attempts`, code de sortie ≠ 0.
3. **AC3** — Après AC2, le cache ne contient ni `go<version>.<arch>.<ext>` ni fichier temporaire `.part`.
4. **AC4** — Le timeout d'inactivité de production vaut 60 s (constante vérifiée par test). Avec un timeout injecté de 50 ms, un serveur qui envoie un octet toutes les 20 ms pendant 300 ms réussit en 1 seule requête ; un serveur silencieux pendant 100 ms déclenche une nouvelle tentative.
5. **AC5** — Réponse 404 : 1 seule requête, message « no version go<X> found ». Réponse 500 ou 429 : message mentionnant le statut HTTP, pas « no version found ».
6. **AC6** — Un fichier partiel n'apparaît jamais sous le nom final : écriture dans `<nom>.part`, puis `rename` après réception complète. Si `Content-Length` est connu et que la taille reçue diffère, la tentative est traitée comme échouée.
7. **AC7** — Réponse sans `Content-Length` : le fichier est téléchargé et installé, aucun panic.
8. **AC8** — Un fichier de cache préexistant plus grand que la nouvelle archive est remplacé intégralement (taille finale = taille servie).
9. **AC9** — `govm install`, `govm use` sans argument, et `govm install abcdef` : erreur lisible, aucun panic.
10. **AC10** — Extraction en échec : le dossier `versions/go/go<version>` est supprimé ; l'archive invalide est supprimée du cache.
11. **AC11** — Ctrl-C pendant le téléchargement : arrêt immédiat, pas de nouvelle tentative, fichier `.part` supprimé.
12. **AC12** — `govm update` utilise le même téléchargeur : AC1 à AC3 valent pour le binaire `govm`.
13. **AC13** — `go vet ./...` et `go test ./...` passent sans erreur.

## 4. Cas limites

- **Erreurs retentées** : timeout, coupure de connexion, EOF prématuré, 5xx, 429. **Non retentées** : 404 et autres 4xx, annulation utilisateur, erreur disque.
- **Disque plein / permission refusée** : pas de retry, `.part` supprimé, erreur d'origine affichée.
- **Deux `govm install` simultanés de la même version** : nom `.part` unique par processus, `rename` atomique ; le dernier gagne, aucun fichier mélangé.
- **Fichier corrompu laissé par une ancienne version de govm** : `install` retélécharge et remplace toujours ; pas de migration.
- **Hors ligne / échec DNS** : 3 tentatives rapides puis erreur ; pause entre tentatives courte (1 s, 2 s).
- **Windows** : `rename` vers un fichier existant et suppression d'un fichier ouvert — fermer avant de supprimer ou renommer.
- **Barre de progression** : réinitialisée à chaque tentative, message `Retrying (2/3)…` affiché.

## 5. Déploiement

Pas de flag : correctif livré dans une version patch via le workflow de release existant. Retour arrière : `govm update` vers le tag précédent ou réinstallation par `scripts/install.sh`. Aucune migration de données ; le format du cache est inchangé.

## 6. Questions ouvertes

Aucune. Décisions prises par Emmanuel le 2026-11-12 :

| # | Décision |
|---|---|
| Q1 | Timeout d'inactivité : une tentative échoue si aucun octet n'est reçu pendant 60 s. Pas de limite sur la durée totale. |
| Q2 | Périmètre complet : B1 à B9, `govm update` inclus. |
| Q3 | Pas de vérification SHA-256 dans ce correctif ; spec séparée plus tard. |
