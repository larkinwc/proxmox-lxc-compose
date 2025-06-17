#!/bin/bash

# Environment Setup for LXC-Compose Integration Testing
# This script helps configure SSH testing environment variables

set -euo pipefail

echo "🔧 LXC-Compose SSH Testing Environment Setup"
echo "============================================="
echo ""

# Function to validate IP address format
validate_ip() {
    local ip=$1
    if [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        IFS='.' read -ra ADDR <<< "$ip"
        for i in "${ADDR[@]}"; do
            if [[ $i -gt 255 ]]; then
                return 1
            fi
        done
        return 0
    else
        # Allow hostnames
        if [[ $ip =~ ^[a-zA-Z0-9.-]+$ ]]; then
            return 0
        fi
        return 1
    fi
}

# Check if environment variables are already set
if [[ -n "${LXC_COMPOSE_REMOTE_HOST:-}" ]] && \
   [[ -n "${LXC_COMPOSE_REMOTE_USER:-}" ]] && \
   [[ -n "${LXC_COMPOSE_SSH_KEY:-}" ]]; then
    echo "✅ Environment variables already configured:"
    echo "   Host: $LXC_COMPOSE_REMOTE_HOST"
    echo "   User: $LXC_COMPOSE_REMOTE_USER"
    echo "   Key:  $LXC_COMPOSE_SSH_KEY"
    echo ""
    echo "To reconfigure, unset variables first:"
    echo "   unset LXC_COMPOSE_REMOTE_HOST LXC_COMPOSE_REMOTE_USER LXC_COMPOSE_SSH_KEY"
    exit 0
fi

echo "This script will help you configure environment variables for SSH testing."
echo "The variables will be saved to ~/.lxc-compose-env for future use."
echo ""

# Collect host information
while true; do
    read -p "🌐 Enter Proxmox/LXC host IP or hostname: " host
    if validate_ip "$host"; then
        break
    else
        echo "❌ Invalid IP address or hostname format. Please try again."
    fi
done

# Collect username
default_user="root"
read -p "👤 Enter SSH username [$default_user]: " user
user=${user:-$default_user}

# Find SSH keys
echo ""
echo "🔍 Looking for SSH keys..."
ssh_keys=()
for key_path in ~/.ssh/id_rsa ~/.ssh/id_ed25519 ~/.ssh/id_ecdsa ~/.ssh/id_dsa; do
    if [[ -f "$key_path" ]]; then
        ssh_keys+=("$key_path")
        echo "   Found: $key_path"
    fi
done

if [[ ${#ssh_keys[@]} -eq 0 ]]; then
    echo "❌ No SSH keys found in ~/.ssh/"
    echo "   Generate one with: ssh-keygen -t ed25519 -C 'your_email@example.com'"
    read -p "🔑 Enter SSH private key path: " ssh_key
else
    echo ""
    echo "🔑 Available SSH keys:"
    for i in "${!ssh_keys[@]}"; do
        echo "   $((i+1))) ${ssh_keys[$i]}"
    done
    echo "   $((${#ssh_keys[@]}+1))) Enter custom path"
    
    while true; do
        read -p "Select SSH key (1-$((${#ssh_keys[@]}+1))): " choice
        if [[ "$choice" =~ ^[0-9]+$ ]] && [[ "$choice" -ge 1 ]] && [[ "$choice" -le $((${#ssh_keys[@]}+1)) ]]; then
            if [[ "$choice" -eq $((${#ssh_keys[@]}+1)) ]]; then
                read -p "🔑 Enter SSH private key path: " ssh_key
            else
                ssh_key="${ssh_keys[$((choice-1))]}"
            fi
            break
        else
            echo "❌ Invalid selection. Please try again."
        fi
    done
fi

# Validate SSH key exists
if [[ ! -f "$ssh_key" ]]; then
    echo "❌ SSH key not found: $ssh_key"
    exit 1
fi

echo ""
echo "📝 Configuration Summary:"
echo "   Host: $host"
echo "   User: $user"
echo "   Key:  $ssh_key"
echo ""

# Test SSH connection
echo "🧪 Testing SSH connection..."
if ssh -i "$ssh_key" -o ConnectTimeout=10 -o BatchMode=yes "$user@$host" 'echo "SSH connection successful"' 2>/dev/null; then
    echo "✅ SSH connection test successful!"
else
    echo "⚠️  SSH connection test failed. Please check:"
    echo "   - Host is reachable: ping $host"
    echo "   - SSH service is running: nc -zv $host 22"
    echo "   - SSH key is authorized: ssh-copy-id -i $ssh_key $user@$host"
    echo ""
    read -p "Continue anyway? (y/N): " continue_anyway
    if [[ ! "$continue_anyway" =~ ^[Yy]$ ]]; then
        echo "❌ Setup cancelled."
        exit 1
    fi
fi

# Save environment variables
env_file="$HOME/.lxc-compose-env"
cat > "$env_file" << EOF
# LXC-Compose SSH Testing Environment Variables
# Generated on $(date)
export LXC_COMPOSE_REMOTE_HOST="$host"
export LXC_COMPOSE_REMOTE_USER="$user"
export LXC_COMPOSE_SSH_KEY="$ssh_key"
EOF

echo ""
echo "✅ Environment configuration saved to: $env_file"
echo ""
echo "🚀 To use these settings:"
echo "   source ~/.lxc-compose-env"
echo "   ./ssh-runner/enhanced-ssh-test.sh"
echo ""
echo "🔧 To add to your shell profile (automatic loading):"
echo "   echo 'source ~/.lxc-compose-env' >> ~/.bashrc"
echo "   source ~/.bashrc"
echo ""

# Offer to source immediately
read -p "Source environment variables now? (Y/n): " source_now
if [[ ! "$source_now" =~ ^[Nn]$ ]]; then
    echo "Setting environment variables for current session..."
    export LXC_COMPOSE_REMOTE_HOST="$host"
    export LXC_COMPOSE_REMOTE_USER="$user"
    export LXC_COMPOSE_SSH_KEY="$ssh_key"
    echo "✅ Environment variables set!"
    echo ""
    echo "🎯 Ready to run tests:"
    echo "   ./ssh-runner/enhanced-ssh-test.sh"
    echo "   ./quick-test.sh"
fi 