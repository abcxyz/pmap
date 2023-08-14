# Release

**pmap is not an official Google product.**

We leverage [goreleaser](https://goreleaser.com/) for both container image
release and SCM (GitHub) release. Due to `goreleaser` limitation, we have to
split the two releases into two config files:

-   `.goreleaser.docker.yaml` for container image release
-   `.goreleaser.yaml` (default) for SCM (GitHub) release

## New Release

-   Send a PR to update all dependencies For Go, 
```sh 
go get -u && go mod tidy
```
-   Sync to main/HEAD 
```sh 
# Checkout to the main/HEAD 
git checkout main
```
-   Create a tag using semantic versioning and then push the tag.
    -   If there are breaking changes, bump the major version
    -   If there are new major features (but not breaking), bump the minor
        version
    -   Nothing important, bump the patch version.
    -   Feel free to use suffixes -alpha, -beta and -rc as needed 

```sh
REL_VER=v0.0.x
# Tag
git tag -f -a $REL_VER -m $REL_VER
# Push tag. This will trigger the release workflow.
git push origin $REL_VER 
``` 
The new pushed tag should trigger the release
workflow which typically does two things: 
- Create a GitHub release with
artifacts (e.g. code zip, binaries, etc.). Note: Goreleaser will automatically
use the git change log to fill the release note. 
- Test, build and upload
artifacts to Artifact Registry (e.g. container images etc.).

## Manually Release Images

Or if you want to build/push images for local development.

```sh
# Set the container registry for the images, for example:
CONTAINER_REGISTRY=us-docker.pkg.dev/my-project/images

# goreleaser expects a "clean" repo to release so commit any local changes if
# needed.
git add . && git commit -m "local changes"

# goreleaser expects a tag.
# The tag must be a semantic version https://semver.org/
# DON'T push the tag if you're not releasing.
git tag -f -a v0.0.0-$(git rev-parse --short HEAD)

# goreleaser will tag the image with the git tag, optionally, override it by:
DOCKER_TAG=mytag

# Use goreleaser to build the images.
# It should in the end push all the images to the given container registry.
# All the images will be tagged with the git tag given earlier.
goreleaser release -f .goreleaser.docker.yaml --clean
```