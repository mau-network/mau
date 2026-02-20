#!/bin/bash
# Test script to verify the GUI POC is working

echo "🧪 Testing Mau GUI POC"
echo "======================"
echo ""

# Check binary exists
if [ ! -f "./mau-gui" ]; then
    echo "❌ Binary not found. Run: go build -o mau-gui"
    exit 1
fi

echo "✅ Binary exists ($(ls -lh mau-gui | awk '{print $5}'))"

# Check it's executable
if [ ! -x "./mau-gui" ]; then
    echo "❌ Binary not executable"
    exit 1
fi

echo "✅ Binary is executable"

# Check help works
if ./mau-gui --help 2>&1 | grep -q "Help Options"; then
    echo "✅ Help flag works"
else
    echo "❌ Help flag failed"
    exit 1
fi

# Check dependencies
echo ""
echo "📦 Checking system dependencies..."
pkg-config --exists gtk4 && echo "✅ GTK4 found: $(pkg-config --modversion gtk4)" || echo "❌ GTK4 missing"
pkg-config --exists libadwaita-1 && echo "✅ Libadwaita found: $(pkg-config --modversion libadwaita-1)" || echo "❌ Libadwaita missing"
pkg-config --exists gobject-introspection-1.0 && echo "✅ GObject Introspection found" || echo "❌ GObject Introspection missing"

echo ""
echo "🎯 POC Status: READY"
echo ""
echo "To run the GUI:"
echo "  ./mau-gui"
echo ""
echo "Note: Requires a display server (X11/Wayland)"
echo "      Use Xvfb for headless testing if needed"
