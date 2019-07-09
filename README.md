# harbor-cleaner
Clean images in Harbor by policies.

## Quick Start 

### Generate Image

```bash
$ make image
```

### Configure

```yaml
host: https://dev.cargo.io
auth:
  user: admin
  password: Pwd123456
policy:
  includePublic: true
  numberPolicy:
    number: 1
  retainTags:
    - v1.*
```

### DryRun

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    harbor-cleaner:v0.0.1 --dryrun=true
```

### Clean

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    harbor-cleaner:v0.0.1
```