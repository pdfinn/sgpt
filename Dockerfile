FROM alpine:3.19

# Copy the pre-built binary
COPY sgpt /usr/local/bin/sgpt

# Run as non-root user for better security
RUN addgroup -S sgpt && adduser -S sgpt -G sgpt
USER sgpt

# Document the expected environment variables
ENV SGPT_API_KEY=""
ENV SGPT_MODEL=""
ENV SGPT_INSTRUCTION=""
ENV SGPT_PROVIDER="openai"

# Create config directory
RUN mkdir -p /home/sgpt/.config/sgpt

# Default command
ENTRYPOINT ["/usr/local/bin/sgpt"] 