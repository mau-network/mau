.PHONY: all build run clean install-deps

all: build

# Install system dependencies (Ubuntu/Debian)
install-deps:
	@echo "Installing GTK4 and Libadwaita..."
	sudo apt install -y libgtk-4-dev libadwaita-1-dev

# Download Go dependencies
deps:
	go mod download
	go mod tidy

# Build the application
build: deps
	@echo "Building Mau GUI POC..."
	go build -o mau-gui

# Run the application
run: build
	./mau-gui

# Clean build artifacts
clean:
	rm -f mau-gui
	rm -rf ~/.mau-gui-poc

# Clean and rebuild
rebuild: clean build

# Run with verbose logging
debug: build
	./mau-gui 2>&1 | tee debug.log
