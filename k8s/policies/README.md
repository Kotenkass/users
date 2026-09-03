# Image security controls

This directory contains Kyverno policies for container image security.

## Policy order

1. `restrict-container-images.yaml`
   - blocks `:latest` and untagged images
   - requires images from the approved registry
   - starts in `audit` mode

2. `require-safe-pod-security-context.yaml`
   - blocks `privileged: true`
   - blocks `runAsNonRoot: false`
   - starts in `audit` mode

3. `require-cosign-signature-prod.yaml`
   - requires production images to be signed with the configured cosign public key
   - starts in `audit` mode
   - replace `REPLACE_ME_WITH_YOUR_COSIGN_PUBLIC_KEY` with your real key before enabling enforcement

## Enforcement

Keep `validationFailureAction: audit` until the existing stack passes all checks.
Only then switch the relevant policies to `enforce`.
