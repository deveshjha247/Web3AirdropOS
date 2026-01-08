FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99

# Install dependencies
RUN apt-get update && apt-get install -y \
    chromium-browser \
    chromium-chromedriver \
    xvfb \
    x11vnc \
    fluxbox \
    supervisor \
    python3 \
    python3-pip \
    curl \
    wget \
    dbus \
    fonts-liberation \
    libnss3 \
    libnspr4 \
    libasound2 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libgbm1 \
    libgtk-3-0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
RUN pip3 install redis websockets aiohttp

WORKDIR /app

# Create directories
RUN mkdir -p /app/profiles /app/logs

# Copy entrypoint script
COPY entrypoint.sh /app/
COPY supervisor.conf /etc/supervisor/conf.d/browser.conf
COPY browser_controller.py /app/

RUN chmod +x /app/entrypoint.sh

# Expose ports
EXPOSE 5900 9222

# Start supervisor
CMD ["/app/entrypoint.sh"]
