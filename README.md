# termux-qemu-docker

A lightweight, profile-aware container VM manager for Termux. `termux-qemu-docker` automates the complex setup required to run Docker containers on Android without root, by spawning an isolated Alpine Linux VM via QEMU and exposing the Docker daemon via a file socket to your Termux environment.
The idea is to provide a similar UX to [lima](github.com/lima-vm/lima) and [colima](github.com/abiosoft/colima) but for Android Termux.

## 🚀 Features

- **Automated Setup**: One-command dependency installation and configuration.
- **Automatic port forwarding**: Detect open ports on the Docker VM and automatically setup port-forwarding
- **True Docker Support**: Run real Docker containers within a lightweight Alpine VM.
- **Profile Aware**: Create multiple isolated VM instances (e.g., `dev`, `prod`, `test`) with unique network ports.
- **Cloud-Init Integration**: Automatic root password and SSH configuration on first boot.
- **Folder Sync**: High-performance host directory sharing via Virtio-9p.

## 📋 Prerequisites

Before starting, ensure you have a modern Android device with Termux installed. The tool will automatically attempt to install the following via `pkg`:
- `qemu-system-aarch64-headless` (or `x86_64`)
- `qemu-utils`
- `openssh`
- `libisoburn` (xorrisofs)
- `dosfstools`

## 🛠️ Installation & Setup

1. **Build the binary**:
   ```bash
   go build -o termux-qemu-docker main.go
   ```

2. **Run the automated setup**:
   This installs dependencies and generates a default `config.yaml`.
   ```bash
   termux-qemu-docker setup
   ```

3. **Start the VM**:
   ```bash
   termux-qemu-docker start
   ```

## 🐳 Using Docker

Once the VM is "Healthy", `termux-qemu-docker` will provide an export command. To connect your Termux `docker` CLI to the VM's daemon, run:

```bash
export DOCKER_HOST=unix://${HOME}/.termux-qemu-docker/docker-default.sock
```

You can now use docker as if it were native:
```bash
docker run --rm hello-world
docker ps
```

## 📂 Folder Sharing

By default, the tool maps your entire `$HOME` directory to the same path inside the VM using **Virtio-9p**. This enables **Consistent Path Mapping**: a file at `/data/data/com.termux/files/home/project/main.go` in Termux is accessible at the exact same path inside the VM.

This allows you to edit code in Termux (using Neovim, Micro, etc.) and run it inside a Docker container with near-native performance and no path-mapping confusion.

### Custom Mounts
You can configure multiple shared folders in your profile's `config.yaml`:

```yaml
mounts:
  - /data/data/com.termux/files/home
  - /sdcard/Documents
```

The tool also automatically attempts to mount the Termux `$PREFIX/tmp` directory if it exists.

## 🔧 Advanced Usage

### Profile Management
Create separate environments using the `-p` or `--profile` flag:
```bash
termux-qemu-docker -p web-dev start
termux-qemu-docker -p database start
termux-qemu-docker list
```
Each profile gets its own disk image and unique SSH port.

### Configuration Overrides
You can override resources during start, and they will be saved to your profile's config:
```bash
termux-qemu-docker start --cpus 4 --memory 4096 --disk 20
```

## 🛑 Stopping & Deleting

To gracefully shut down the VM:
```bash
termux-qemu-docker stop
# Or for a specific profile
termux-qemu-docker -p web-dev stop
```

To completely remove a profile and its disk image:
```bash
termux-qemu-docker delete
# Or for a specific profile
termux-qemu-docker -p web-dev delete
```

## 🛡️ Security
- **Isolation**: Containers run inside a dedicated VM, providing a layer of security between Docker and your Android OS.
- **Encapsulation**: Remote commands use escaped shell arguments to prevent injection.
- **Lifecycle**: Resource allocation and port listeners are context-bound and shut down automatically with the VM.
