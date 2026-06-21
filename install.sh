#!/bin/bash

# P-BOX Linux One-Click Installation Script
# https://github.com/star8618/P-BOX

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
INSTALL_DIR="/etc/p-box"
SERVICE_NAME="p-box"
DEFAULT_PORT=8666
GITHUB_REPO="star8618/P-BOX"
GITHUB_API="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"

echo -e "${CYAN}"
echo "========================================"
echo "      P-BOX Linux Installer"
echo "========================================"
echo -e "${NC}"

# Check root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}ERROR: Please run as root (sudo)${NC}"
    exit 1
fi

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l|armv7)
            echo "armv7"
            ;;
        *)
            echo ""
            ;;
    esac
}

ARCH=$(detect_arch)
if [ -z "$ARCH" ]; then
    echo -e "${RED}ERROR: Unsupported architecture: $(uname -m)${NC}"
    exit 1
fi
echo -e "${GREEN}Detected architecture: ${CYAN}${ARCH}${NC}"

# Get latest version
echo -e "${BLUE}Fetching latest version...${NC}"

VERSION=""
if command -v curl &> /dev/null; then
    VERSION=$(curl -s "$GITHUB_API" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' | head -n 1)
elif command -v wget &> /dev/null; then
    VERSION=$(wget -qO- "$GITHUB_API" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' | head -n 1)
fi

# Remove 'v' prefix if present
VERSION=${VERSION#v}

if [ -z "$VERSION" ]; then
    echo -e "${RED}ERROR: Failed to get latest version${NC}"
    exit 1
fi

echo -e "${GREEN}Latest version: ${CYAN}v${VERSION}${NC}"

# Download URL (GitHub uses 'v' prefix in release tags)
FILENAME="p-box-${VERSION}-linux-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${FILENAME}"
CDN_URL="https://ghfast.top/${DOWNLOAD_URL}"

echo -e "${BLUE}Downloading P-BOX...${NC}"

TEMP_DIR=$(mktemp -d)
TEMP_FILE="${TEMP_DIR}/${FILENAME}"
download_success=false

if curl -sL --connect-timeout 15 -o "$TEMP_FILE" "$CDN_URL" 2>/dev/null; then
    if [ -s "$TEMP_FILE" ] && file "$TEMP_FILE" | grep -q "gzip"; then
        echo -e "${GREEN}Downloaded from CDN${NC}"
        download_success=true
    else
        rm -f "$TEMP_FILE"
    fi
fi

if [ "$download_success" = false ]; then
    echo -e "${YELLOW}CDN failed, trying GitHub...${NC}"
    if curl -sL --connect-timeout 30 -o "$TEMP_FILE" "$DOWNLOAD_URL" 2>/dev/null; then
        if [ -s "$TEMP_FILE" ] && file "$TEMP_FILE" | grep -q "gzip"; then
            echo -e "${GREEN}Downloaded from GitHub${NC}"
            download_success=true
        fi
    fi
fi

if [ "$download_success" = false ]; then
    echo -e "${RED}ERROR: Download failed${NC}"
    rm -rf "$TEMP_DIR"
    exit 1
fi

if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    echo -e "${YELLOW}Stopping existing service...${NC}"
    systemctl stop "$SERVICE_NAME"
fi

echo -e "${BLUE}Installing to ${INSTALL_DIR}...${NC}"
mkdir -p "$INSTALL_DIR"

tar -xzf "$TEMP_FILE" -C "$TEMP_DIR"

EXTRACTED_DIR=$(find "$TEMP_DIR" -maxdepth 1 -type d -name "p-box-*" | head -n 1)
if [ -z "$EXTRACTED_DIR" ]; then
    EXTRACTED_DIR="$TEMP_DIR"
fi

if [ -d "$EXTRACTED_DIR" ] && [ "$(ls -A "$EXTRACTED_DIR")" ]; then
    cp -r "$EXTRACTED_DIR"/* "$INSTALL_DIR/"
else
    echo -e "${RED}ERROR: Extraction failed${NC}"
    rm -rf "$TEMP_DIR"
    exit 1
fi

chmod +x "$INSTALL_DIR/p-box"
chmod +x "$INSTALL_DIR/install-nginx.sh" 2>/dev/null || true

if [ -f "$INSTALL_DIR/config.yaml" ]; then
    sed -i "s/port: 8383/port: ${DEFAULT_PORT}/" "$INSTALL_DIR/config.yaml"
    echo -e "${GREEN}Updated default port to ${DEFAULT_PORT}${NC}"
fi

rm -rf "$TEMP_DIR"
echo -e "${GREEN}Installation complete${NC}"

echo -e "${BLUE}Creating systemd service...${NC}"

cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=P-BOX Proxy Management Panel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/p-box
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

prepare_transparent_gateway() {
    echo -e "${BLUE}Preparing transparent gateway mode...${NC}"

    modprobe tun 2>/dev/null || true
    sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true
    sysctl -w net.ipv6.conf.all.forwarding=1 >/dev/null 2>&1 || true

    cat > /usr/local/sbin/p-box-docker-dns-proxy.sh << 'EOF'
#!/bin/sh
set -eu

DNS_PORT="${PBOX_DNS_PORT:-1053}"
IPTABLES="$(command -v iptables || true)"

if [ -z "$IPTABLES" ]; then
    exit 0
fi

for i in 1 2 3 4 5 6 7 8 9 10; do
    if ip link show mihomo >/dev/null 2>&1; then
        ip route replace 198.18.0.0/16 dev mihomo 2>/dev/null || true
        if command -v resolvectl >/dev/null 2>&1; then
            resolvectl revert mihomo 2>/dev/null || true
            resolvectl default-route mihomo false 2>/dev/null || true
        fi
        break
    fi
    sleep 1
done

{ ip -o link show 2>/dev/null | awk -F': ' '{print $2}' | grep -E '^(docker0|br-|podman|cni|nerdctl)' || true; } | while read -r iface; do
    [ -n "$iface" ] || continue
    "$IPTABLES" -C INPUT -i "$iface" -p udp --dport "$DNS_PORT" -j ACCEPT 2>/dev/null || "$IPTABLES" -I INPUT 1 -i "$iface" -p udp --dport "$DNS_PORT" -j ACCEPT || true
    "$IPTABLES" -C INPUT -i "$iface" -p tcp --dport "$DNS_PORT" -j ACCEPT 2>/dev/null || "$IPTABLES" -I INPUT 1 -i "$iface" -p tcp --dport "$DNS_PORT" -j ACCEPT || true
    "$IPTABLES" -C INPUT -i "$iface" -p tcp --dport 7890 -j ACCEPT 2>/dev/null || "$IPTABLES" -I INPUT 1 -i "$iface" -p tcp --dport 7890 -j ACCEPT || true
    "$IPTABLES" -C INPUT -i "$iface" -p udp --dport 7890 -j ACCEPT 2>/dev/null || "$IPTABLES" -I INPUT 1 -i "$iface" -p udp --dport 7890 -j ACCEPT || true
    "$IPTABLES" -C INPUT -i "$iface" -p tcp --dport 7892 -j ACCEPT 2>/dev/null || "$IPTABLES" -I INPUT 1 -i "$iface" -p tcp --dport 7892 -j ACCEPT || true
    "$IPTABLES" -C INPUT -i "$iface" -p udp --dport 7892 -j ACCEPT 2>/dev/null || "$IPTABLES" -I INPUT 1 -i "$iface" -p udp --dport 7892 -j ACCEPT || true
    "$IPTABLES" -C FORWARD -i "$iface" -o mihomo -j ACCEPT 2>/dev/null || "$IPTABLES" -I FORWARD 1 -i "$iface" -o mihomo -j ACCEPT || true
    "$IPTABLES" -C FORWARD -i mihomo -o "$iface" -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || "$IPTABLES" -I FORWARD 1 -i mihomo -o "$iface" -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT || true
    while "$IPTABLES" -t nat -D PREROUTING -i "$iface" -p udp --dport 53 -j REDIRECT --to-ports 53 2>/dev/null; do :; done
    while "$IPTABLES" -t nat -D PREROUTING -i "$iface" -p tcp --dport 53 -j REDIRECT --to-ports 53 2>/dev/null; do :; done
    while "$IPTABLES" -t nat -D PREROUTING -i "$iface" -p udp --dport 53 -j REDIRECT --to-ports 1053 2>/dev/null; do :; done
    while "$IPTABLES" -t nat -D PREROUTING -i "$iface" -p tcp --dport 53 -j REDIRECT --to-ports 1053 2>/dev/null; do :; done
    while "$IPTABLES" -t nat -D PREROUTING -i "$iface" -d 198.18.0.1/32 -j RETURN 2>/dev/null; do :; done
    while "$IPTABLES" -t nat -D PREROUTING -i "$iface" -p tcp -d 198.18.0.0/16 -j REDIRECT --to-ports 7892 2>/dev/null; do :; done
    "$IPTABLES" -t nat -A PREROUTING -i "$iface" -p udp --dport 53 -j REDIRECT --to-ports "$DNS_PORT" || true
    "$IPTABLES" -t nat -A PREROUTING -i "$iface" -p tcp --dport 53 -j REDIRECT --to-ports "$DNS_PORT" || true
    "$IPTABLES" -t nat -A PREROUTING -i "$iface" -d 198.18.0.1/32 -j RETURN || true
    "$IPTABLES" -t nat -A PREROUTING -i "$iface" -p tcp -d 198.18.0.0/16 -j REDIRECT --to-ports 7892 || true
done
EOF
    chmod +x /usr/local/sbin/p-box-docker-dns-proxy.sh

    cat > /etc/systemd/system/p-box-docker-dns-proxy.service << 'EOF'
[Unit]
Description=P-BOX Docker DNS transparent proxy rules
After=network-online.target docker.service podman.service p-box.service
Wants=network-online.target p-box.service

[Service]
Type=oneshot
Environment=PBOX_DNS_PORT=1053
ExecStart=/usr/local/sbin/p-box-docker-dns-proxy.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF
}

prepare_transparent_gateway

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl enable p-box-docker-dns-proxy.service 2>/dev/null || true
echo -e "${GREEN}Service enabled for auto-start${NC}"

if [ -f "$INSTALL_DIR/install-nginx.sh" ]; then
    echo -e "${BLUE}Running Nginx installation script...${NC}"
    chmod +x "$INSTALL_DIR/install-nginx.sh"
    cd "$INSTALL_DIR" && bash ./install-nginx.sh || echo -e "${YELLOW}Nginx script completed with warnings${NC}"
fi

echo -e "${BLUE}Starting P-BOX service...${NC}"
systemctl start "$SERVICE_NAME"
systemctl start p-box-docker-dns-proxy.service 2>/dev/null || true

sleep 2
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo -e "${GREEN}P-BOX is running${NC}"
else
    echo -e "${YELLOW}Service may need manual start: systemctl start ${SERVICE_NAME}${NC}"
fi

IP_ADDR=$(hostname -I | awk '{print $1}' 2>/dev/null || echo "localhost")

echo ""
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}      P-BOX Installation Complete       ${NC}"
echo -e "${CYAN}========================================${NC}"
echo -e "Install Path: ${GREEN}${INSTALL_DIR}${NC}"
echo -e "Web Panel:    ${GREEN}http://${IP_ADDR}:${DEFAULT_PORT}${NC}"
echo ""
echo "Commands:"
echo "  systemctl start ${SERVICE_NAME}"
echo "  systemctl stop ${SERVICE_NAME}"
echo "  systemctl restart ${SERVICE_NAME}"
echo "  systemctl status ${SERVICE_NAME}"
echo ""
