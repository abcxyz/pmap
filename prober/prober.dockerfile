# Base image that supports bash
FROM gcr.io/distroless/static

COPY pmap-prober /pmap-prober

# Run the binary on container startup.
ENTRYPOINT ["./pmap-prober"]
