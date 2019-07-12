# Harbor Cleaner
Clean images in Harbor by policies.

## Policies

### Tag Number Policy

Tag number policy specifies how many tags to retain for each repo, and clean other old tags.

Note when delete a tag, Docker will delete by underneath digest ID actually. So if in the same repository, there are other tags that share the same image digest (that's to say, images given by these tags are the same), they will also be deleted.

In this cleaner, this situation is considered, when to delete a tag, if it will affect tags that should be retained, then the tag will also be retained.

For example, if the policy wants to retain 3 tag per repository, but there is a repository with following tags:

- v0.4   (Digest D)
- v0.3   (Digest C)
- v0.2.2 (Digest B)
- v0.2.1 (Digest B)
- v0.1   (Digest A)

To retain only latest 3 tags, tags `v0.2.1`, `v0.1` will be deleted. But as `v0.2.2` has same digest ID to `v0.2.1`, `v0.2.2` will also be deleted if we are to delete `v0.2.1`. So finally, `v0.2.1` will be kept, resulting in following tags:

- v0.4   (Digest D)
- v0.3   (Digest C)
- v0.2.2 (Digest B)
- v0.2.1 (Digest B)

## How To Use

### Get Image

```bash
$ make image
```

You can also pull one from registry.

```bash
$ docker pull k8sdevops/harbor-cleaner:v0.0.3
```

### Configure

```yaml
host: https://dev.cargo.io
version: 1.7
auth:
  user: admin
  password: Pwd123456
projects:
  - projectA
policy:
  includePublic: true
  numberPolicy:
    number: 5
  retainTags:
    - v1.0
```

- `projects` defines which projects to clean images for, if want to clean all projects, make it empty by remove it from configure file.
- `retainTags` defines tags that must be kept, `?`, `*` supported. For example, `v1.*`. Remove this configure if you don't want to use it.

### DryRun

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    harbor-cleaner:v0.0.3 --dryrun=true
```

### Clean

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    harbor-cleaner:v0.0.3
```

# Supported Version

- 1.4.x
- 1.5.x
- 1.6.x
- 1.7.x
- 1.8.x

It may work for other versions, but this is not tested.