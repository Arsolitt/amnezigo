# Release image for amnezigo, built by GoReleaser (dockers_v2).
# The binary is already compiled per platform, so the build context contains
# <os>/<arch>/amnezigo plus the license files listed in extra_files — the source
# tree is NOT part of the context.
# The base tag is pinned because amnezigo targets the AmneziaWG 3.1 userspace
# engine; the tag is amd64-only, matching the single platform published here.
# There is no USER directive: amnezigo is a CLI that reads and writes configs in
# the mounted working directory, so it must run as the invoking host user (pass
# `--user $(id -u):$(id -g)`); fixed-image stages here would only break that.
FROM amneziavpn/amneziawg-go:3.1.20260828

# TARGETPLATFORM is a buildx predefined ARG. It must be re-declared inside the
# stage: a global ARG (before FROM) is not in scope for instructions after FROM,
# and without this the COPY below expands to an empty platform path.
ARG TARGETPLATFORM

# No `apk add` layer: the base image is Alpine 3.19 and already ships bash
# (/bin/bash) and the ca-certificates bundle (/etc/ssl/certs/ca-certificates.crt),
# so releases stay buildable without network access beyond the base pull.

COPY $TARGETPLATFORM/amnezigo /usr/local/bin/amnezigo
COPY NOTICE LICENSE /usr/share/doc/amnezigo/
COPY licenses /usr/share/doc/amnezigo/licenses/

ENTRYPOINT ["/usr/local/bin/amnezigo"]
