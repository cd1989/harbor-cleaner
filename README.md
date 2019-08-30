# Harbor Cleaner

Clean images in Harbor by policies.

## Features

- **Delete tags without side effects** As we known when we delete a tag from a repo in docker registry, the underneath manifest is deleted, so are other tags what share the same manifest. In this tool, we protect tags from such situation.
- **Delete by policies** Support delete tags by configurable policies
- **Dry run before actual cleanup** To see what would be cleaned up before performing real cleanup.

## Policies

### Retain N Tags

This policy specifies how many tags to retain for each repo, and clean other old tags.

## How To Use

### Get Image

```bash
$ make image VERSION=latest
```

You can also pull one from DockerHub.

```bash
$ docker pull k8sdevops/harbor-cleaner:v0.1.0
```

### Configure

```yaml
# Host of the Harbor
host: https://dev.cargo.io
# Version of the Harbor, e.g. 1.7, 1.4.0
version: 1.7
# Admin account
auth:
  user: admin
  password: Pwd123456
# Projects list to clean images for, it you want to clean images for all
# projects, leave it empty.
projects: []
# Policy to clean images
policy:
  # Policy type, e.g. "number", "recentlyNotTouched"
  type: number
  # Number policy: to retain the latest N tags for each repo
  numberPolicy:
    number: 5
  # Tags that should be retained anyway, '?', '*' supported.
  retainTags: []
```

### DryRun

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    harbor-cleaner:latest --dryrun=true
```

### Clean

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    harbor-cleaner:latest
```

## Supported Version

- 1.4.x
- 1.5.x
- 1.6.x
- 1.7.x
- 1.8.x

It may work for other versions, but this is not tested.