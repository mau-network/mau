#!/bin/sh
set -e

# Mau E2E Test Container Entrypoint

# Configuration from environment variables
DATA_DIR="${MAU_DATA_DIR:-/data/.mau}"
PEER_NAME="${MAU_PEER_NAME:-peer}"
PASSPHRASE="${MAU_PASSPHRASE:-testpass}"
BOOTSTRAP_NODE="${MAU_BOOTSTRAP:-}"

echo "[entrypoint] Starting Mau peer: $PEER_NAME"
echo "[entrypoint] Data directory: $DATA_DIR"

# Ensure data directory exists
mkdir -p "$DATA_DIR"
cd "$DATA_DIR"

# Check if account exists, if not initialize
if [ ! -f "$DATA_DIR/account.key" ]; then
    echo "[entrypoint] Initializing new account..."
    
    # Use expect to handle the interactive prompts
    # Install expect if not present
    which expect > /dev/null || apk add --no-cache expect
    
    # Create expect script
    cat > /tmp/init.exp << 'EXPECT_EOF'
#!/usr/bin/expect -f
set timeout 10
set passphrase [lindex $argv 0]
set peer_name [lindex $argv 1]
set peer_email [lindex $argv 2]

spawn mau init --name $peer_name --email $peer_email

expect "Passphrase:"
send "$passphrase\r"

expect eof
EXPECT_EOF

    chmod +x /tmp/init.exp
    
    # Run init with expect
    /tmp/init.exp "$PASSPHRASE" "$PEER_NAME" "${PEER_NAME}@mau.test" || {
        echo "[entrypoint] Account initialization failed"
        exit 1
    }
    
    rm /tmp/init.exp
    
    echo "[entrypoint] Account initialized successfully"
else
    echo "[entrypoint] Using existing account"
fi

# Execute the requested command
case "$1" in
    serve)
        echo "[entrypoint] Starting Mau server..."
        
        # Start server with expect to handle passphrase
        cat > /tmp/serve.exp << 'EXPECT_EOF'
#!/usr/bin/expect -f
set timeout -1
set passphrase [lindex $argv 0]

spawn mau serve

expect "Passphrase:"
send "$passphrase\r"

# Keep the process running
expect eof
EXPECT_EOF

        chmod +x /tmp/serve.exp
        exec /tmp/serve.exp "$PASSPHRASE"
        ;;
    
    shell)
        echo "[entrypoint] Starting interactive shell..."
        exec /bin/sh
        ;;
    
    *)
        # Pass through any other command
        exec "$@"
        ;;
esac
