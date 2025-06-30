FROM python:3.11-slim

WORKDIR /app

# Copy the webhook proxy script
COPY github-discord-proxy.py .

# Make it executable
RUN chmod +x github-discord-proxy.py

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV PYTHONUNBUFFERED=1

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:8080')"

# Run the proxy
CMD ["python3", "github-discord-proxy.py"]
