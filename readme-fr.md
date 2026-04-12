# Fleet

Un outil de gestion de workspace multi-dépôts pour le workflow GitHub Fork, inspiré de [l'outil `repo` de Google](https://gerrit.googlesource.com/git-repo/).

Comme `repo`, Fleet utilise des fichiers manifest XML pour gérer de façon déclarative plusieurs dépôts Git — mais conçu spécifiquement pour le workflow **GitHub Fork + Pull Request** plutôt que Gerrit. Il prend en charge le clone, la synchronisation, la gestion de branches, le push, la création de PR et l'exécution de commandes sur tous les dépôts en une seule opération.

[English Documentation](README.md)

## Installation

```bash
# Installation en une commande (macOS / Linux)
curl -sSfL https://raw.githubusercontent.com/mingyuans/fleet-cli/main/install.sh | sh

# Ou spécifier une version / répertoire d'installation
FLEET_VERSION=v0.1.0 FLEET_INSTALL_DIR=~/.local/bin \
  curl -sSfL https://raw.githubusercontent.com/mingyuans/fleet-cli/main/install.sh | sh

# Ou construire depuis les sources
git clone https://github.com/mingyuans/fleet-cli.git
cd fleet-cli
make install
```

## Démarrage rapide

**1. Créer un workspace avec `fleet.xml` :**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="git@github.com:my-org/" />
  <default remote="github" revision="master" sync-j="4" />

  <project name="user-service"    path="services/user-service"  groups="core" />
  <project name="order-service"   path="services/order-service" groups="commerce" />
</manifest>
```

**2. Ajouter un fichier personnel `local_fleet.xml` pour votre fork :**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="fork" fetch="git@github.com:your-username/" />
  <default push="fork" />
</manifest>
```

**3. Cloner tous les dépôts :**

```bash
fleet init
```

## Commandes

| Commande | Description |
|----------|-------------|
| `fleet init` | Cloner les dépôts et configurer les remotes |
| `fleet sync` | Récupérer les dernières modifications en amont (rebase sur la branche par défaut, fetch sur les branches feature) |
| `fleet status` | Afficher la branche, l'état (propre/modifié) et l'avance/retard pour tous les dépôts |
| `fleet start <branch>` | Créer une branche feature sur tous les dépôts depuis le HEAD en amont |
| `fleet finish <branch>` | Supprimer une branche et revenir à la branche par défaut (`-r` pour supprimer aussi la branche distante) |
| `fleet push` | Pousser la branche courante vers le fork (`--all` pour inclure la branche par défaut) |
| `fleet pr` | Pousser et créer des PR via le CLI `gh` (`-t` pour définir le titre) |
| `fleet forall -c "cmd"` | Exécuter une commande shell dans tous les dépôts |

Toutes les commandes supportent `-g <expr>` pour filtrer par groupe (`,` = OU, `+` = ET).

## Workflow typique

```bash
fleet init                          # cloner tous les dépôts
fleet sync                          # récupérer les dernières modifications en amont
fleet start feature/my-feature      # créer une branche partout
# ... effectuer les modifications ...
fleet push                          # pousser vers le fork
fleet pr -t "feat: my feature"      # créer les PR
fleet finish feature/my-feature     # nettoyer après la fusion
```

## Référence du Manifest

Fleet utilise deux fichiers XML à la racine du workspace :

| Fichier | Rôle |
|---------|------|
| `fleet.xml` | Configuration partagée de l'équipe (versionné dans Git) |
| `local_fleet.xml` | Surcharges personnelles — remote fork, dépôts supplémentaires (dans .gitignore) |

Lorsque les deux fichiers existent, `local_fleet.xml` est fusionné dans `fleet.xml` :

- **Remotes** — même nom remplace ; nouveaux s'ajoutent
- **Default** — surcharge attribut par attribut
- **Projects** — même nom remplace attribut par attribut ; nouveaux s'ajoutent

Voir [docs/example-fleet.xml](docs/example-fleet.xml) pour un exemple complet annoté.

### Attributs clés

| Élément | Attributs |
|---------|-----------|
| `<remote>` | `name`, `fetch`, `review` |
| `<default>` | `remote`, `revision`, `sync-j`, `push`, `master-main-compat` |
| `<project>` | `name`, `path`, `groups`, `remote`, `revision`, `push` |

Définir `master-main-compat="true"` sur `<default>` pour basculer automatiquement entre `master` et `main`.

## Variables d'environnement

| Variable | Description | Valeur par défaut |
|----------|-------------|-------------------|
| `FLEET_MANIFEST` | Chemin vers le manifest principal | `<workspace>/fleet.xml` |
| `FLEET_LOCAL_MANIFEST` | Chemin vers le manifest local | `<workspace>/local_fleet.xml` |

## Documentation

- [English User Guide](docs/usage-en.md)
- [中文使用说明](docs/usage-zh.md)
- [Exemple de Manifest](docs/example-fleet.xml)

## Licence

MIT
