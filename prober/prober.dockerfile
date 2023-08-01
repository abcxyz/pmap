# Base image that supports bash
FROM cgr.dev/chainguard/bash:latest

COPY pmap-prober /pmap-prober

# Run the binary on container startup.
ENTRYPOINT ["./pmap-prober"]
