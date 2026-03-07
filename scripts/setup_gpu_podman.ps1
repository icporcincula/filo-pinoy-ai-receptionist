# setup_gpu_podman.ps1
# Configures NVIDIA Container Toolkit for Podman on Windows.
#
# Podman Desktop uses its own WSL2 machine (podman-machine-default) which is
# Fedora-based — NOT Ubuntu. This script installs the toolkit using dnf (rpm),
# then generates the CDI spec so containers can see the GPU.
#
# Prerequisites:
#   - Podman Desktop installed (https://podman.io)
#   - NVIDIA Game Ready or Studio driver 525+ on Windows
#
# Run with:
#   PowerShell -ExecutionPolicy Bypass -File .\scripts\setup_gpu_podman.ps1

Write-Host "=== AI Receptionist: GPU Passthrough Setup for Podman ===" -ForegroundColor Cyan

# ---------------------------------------------------------------------------
# Step 1: Confirm the Podman machine is running
# ---------------------------------------------------------------------------
Write-Host "`n[1/4] Checking Podman machine status..." -ForegroundColor Yellow
podman machine info
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Podman machine is not running. Start it with:" -ForegroundColor Red
    Write-Host "  podman machine start" -ForegroundColor White
    exit 1
}

# ---------------------------------------------------------------------------
# Step 2: Install NVIDIA Container Toolkit inside the Podman machine (Fedora)
#
# The Podman machine is Fedora-based so we use the rpm repo, not apt/deb.
# Variables are passed via single-quoted PS string to prevent PS from
# expanding $variables before they reach bash.
# ---------------------------------------------------------------------------
Write-Host "`n[2/4] Installing NVIDIA Container Toolkit (Fedora/rpm)..." -ForegroundColor Yellow

$installScript = @'
set -e
if command -v nvidia-ctk &>/dev/null; then
    echo "[toolkit] nvidia-container-toolkit already installed, skipping."
    nvidia-ctk --version
else
    echo "[toolkit] Adding NVIDIA rpm repo..."
    curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo \
        | sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo

    echo "[toolkit] Installing nvidia-container-toolkit..."
    sudo dnf install -y nvidia-container-toolkit

    echo "[toolkit] Install complete."
    nvidia-ctk --version
fi
'@

podman machine ssh -- bash -c $installScript
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Toolkit installation failed. Check output above." -ForegroundColor Red
    exit 1
}

# ---------------------------------------------------------------------------
# Step 3: Generate CDI spec and configure Podman runtime
#
# CDI (Container Device Interface) is how Podman passes the GPU into
# containers. nvidia-ctk generates /etc/cdi/nvidia.yaml on the Podman
# machine, which is what `--device nvidia.com/gpu=all` resolves against.
# ---------------------------------------------------------------------------
Write-Host "`n[3/4] Generating CDI spec and configuring Podman runtime..." -ForegroundColor Yellow

$configScript = @'
set -e

echo "[cdi] Generating CDI spec..."
sudo mkdir -p /etc/cdi
sudo nvidia-ctk cdi generate --output=/etc/cdi/nvidia.yaml

echo "[cdi] CDI devices available:"
nvidia-ctk cdi list

echo "[runtime] Detecting available container runtime..."
if command -v crun &>/dev/null; then
    RUNTIME=crun
elif command -v runc &>/dev/null; then
    RUNTIME=runc
else
    echo "ERROR: neither crun nor runc found"
    exit 1
fi
echo "[runtime] Using runtime: $RUNTIME"

mkdir -p $HOME/.config/containers
nvidia-ctk runtime configure --runtime=$RUNTIME --config=$HOME/.config/containers/containers.conf

echo "[runtime] containers.conf contents:"
cat $HOME/.config/containers/containers.conf 2>/dev/null || echo "file not created"
'@

podman machine ssh -- bash -c $configScript
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: CDI generation failed." -ForegroundColor Red
    Write-Host "Make sure your NVIDIA Windows driver is 525+ and WSL2 GPU support is enabled." -ForegroundColor Yellow
    exit 1
}

# ---------------------------------------------------------------------------
# Step 4: Test GPU passthrough
# ---------------------------------------------------------------------------
Write-Host "`n[4/4] Testing GPU passthrough (running nvidia-smi in a container)..." -ForegroundColor Yellow
Write-Host "This pulls a small CUDA image on first run..." -ForegroundColor Gray

podman run --rm --device nvidia.com/gpu=all --security-opt=label=disable `
    nvcr.io/nvidia/cuda:12.3.1-base-ubi9 nvidia-smi

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n=== SUCCESS ===" -ForegroundColor Green
    Write-Host "GPU passthrough is working. You can now run:" -ForegroundColor Green
    Write-Host "  podman-compose up -d" -ForegroundColor White
} else {
    Write-Host "`n=== FAILED ===" -ForegroundColor Red
    Write-Host "Common causes:" -ForegroundColor Yellow
    Write-Host "  - NVIDIA driver < 525 on Windows host" -ForegroundColor White
    Write-Host "  - WSL2 GPU integration not enabled in Podman Desktop settings" -ForegroundColor White
    Write-Host "  - CDI spec generation failed silently" -ForegroundColor White
    Write-Host "`nManual debug inside the Podman machine:" -ForegroundColor Yellow
    Write-Host "  podman machine ssh" -ForegroundColor White
    Write-Host "  ls /etc/cdi/          # should contain nvidia.yaml" -ForegroundColor White
    Write-Host "  nvidia-smi            # should show your GPU from inside the machine" -ForegroundColor White
}