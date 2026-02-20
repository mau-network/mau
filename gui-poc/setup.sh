#!/bin/bash
set -e

echo "🛸 Mau GUI POC Setup Script"
echo "============================"
echo ""

# Check for required tools
echo "Checking prerequisites..."
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.26 or later."
    exit 1
fi

echo "✅ Go $(go version | awk '{print $3}')"

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo "❌ Cannot detect OS. Please install GTK4 and Libadwaita manually."
    exit 1
fi

echo "📦 Installing system dependencies for $OS..."

case "$OS" in
    ubuntu|debian|pop)
        sudo apt update
        sudo apt install -y libgtk-4-dev libadwaita-1-dev pkg-config
        ;;
    fedora)
        sudo dnf install -y gtk4-devel libadwaita-devel pkg-config
        ;;
    arch|manjaro)
        sudo pacman -S --noconfirm gtk4 libadwaita pkg-config
        ;;
    *)
        echo "⚠️  Unsupported OS: $OS"
        echo "Please install these packages manually:"
        echo "  - GTK4 development files"
        echo "  - Libadwaita development files"
        echo "  - pkg-config"
        exit 1
        ;;
esac

# Verify installation
echo ""
echo "Verifying installation..."
GTK4_VERSION=$(pkg-config --modversion gtk4 2>/dev/null || echo "not found")
ADW_VERSION=$(pkg-config --modversion libadwaita-1 2>/dev/null || echo "not found")

if [ "$GTK4_VERSION" == "not found" ]; then
    echo "❌ GTK4 installation failed"
    exit 1
fi

if [ "$ADW_VERSION" == "not found" ]; then
    echo "❌ Libadwaita installation failed"
    exit 1
fi

echo "✅ GTK4 $GTK4_VERSION"
echo "✅ Libadwaita $ADW_VERSION"

# Download Go dependencies
echo ""
echo "📥 Downloading Go dependencies..."
go mod download
go mod tidy

# Build
echo ""
echo "🔨 Building Mau GUI POC..."
go build -o mau-gui

echo ""
echo "✨ Setup complete!"
echo ""
echo "To run the application:"
echo "  ./mau-gui"
echo ""
echo "Or use make:"
echo "  make run"
