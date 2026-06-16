package vm

// tiny-cloud doesn't seem to support - | in runcmd
// this is formatted to support both tiny-cloud and cloud-init
// tiny-cloud also doesn't add non existing groups so we add docker explicitly at the end
const userDataTemplate = `#cloud-config
ssh_pwauth: true
hostname: {{.ProfileName}}
chpasswd:
  expire: false
groups:
  - wheel
  - docker
users:
  - name: {{.SSHUser}}
    passwd: "{{.SSHPassword}}"
    lock_passwd: false
    groups: [docker, wheel]
    doas:
      - "permit nopass {{ .SSHUser }} as root"
    shell: /bin/ash
    ssh_authorized_keys:
      - {{.PublicKey}}

write_files:
  - path: /etc/containerd/config.toml
    content: |
      version = 3
      root = "/var/lib/containerd"
      state = "/run/containerd"

      disabled_plugins = [
        # Disables Kubernetes interface
        "io.containerd.grpc.v1.cri",

        # Disables storage drivers unsupported by the VM disk image layout
        "io.containerd.snapshotter.v1.btrfs",
        "io.containerd.snapshotter.v1.zfs",
        "io.containerd.snapshotter.v1.devmapper",
        "io.containerd.snapshotter.v1.aufs",

        # Disables cloud/bare-metal systems that waste CPU cycles inside QEMU
        "io.containerd.internal.v1.tracing",
        "io.containerd.nri.v1.nri",
      ]

      [grpc]
        address = "/run/containerd/containerd.sock"
        tcp_address = ""
        max_recv_message_size = 16777216
        max_send_message_size = 16777216

      [debug]
        level = "info"

      [plugins]
        [plugins."io.containerd.cri.v1.images"]
          snapshotter = "overlayfs"
          [plugins."io.containerd.cri.v1.images".registry]
            config_path = "/etc/containerd/certs.d:/etc/docker/certs.d"

        [plugins."io.containerd.cri.v1.runtime"]
          [plugins."io.containerd.cri.v1.runtime".cni]
            bin_dirs = ["/usr/libexec/cni"]

        [plugins."io.containerd.grpc.v1.cri"]
          stream_server_address = "/run/containerd/containerd-stream.sock"
          stream_server_port = "0"
          disable_tcp_service = true

  - path: /etc/docker/daemon.json
    content: |
      {
        "hosts": ["unix:///var/run/docker.sock"],
        "storage-driver": "overlay2",
        "max-concurrent-downloads": 10,
        "containerd": "/run/containerd/containerd.sock",
        "log-opts": {
           "max-size": "10m",
           "max-file": "2"
         }
      }

# package_update: true
# package_upgrade: true
# package_reboot_if_required: true
# # package_reboot: true
packages:
  - docker
  - docker-compose
  - containerd
  - socat
  - mount
  - linux-virt

mounts:
  - [ swap, null ]
runcmd:
  - 'echo 1 > /proc/sys/net/ipv4/ip_forward'
  - 'echo 1 > /proc/sys/fs/may_detach_mounts'
  - swapoff -a
  - sed -i '/swap/d' /etc/fstab
  - rc-update del swap default
  - 'rc-update add cgroups default'
  - 'rc-service cgroups start'
  - 'rc-update add containerd default'
  - 'rc-service containerd start'
  - 'rc-update add docker default'
  - 'rc-service docker start'
  - 'sed -i "s/#PasswordAuthentication yes/PasswordAuthentication yes/g" /etc/ssh/sshd_config'
  - 'sed -i "s/PasswordAuthentication no/PasswordAuthentication yes/g" /etc/ssh/sshd_config'
  - 'sed -i "s/AllowTcpForwarding no/AllowTcpForwarding yes/g" /etc/ssh/sshd_config'
  - 'grep -q "^AllowTcpForwarding yes" /etc/ssh/sshd_config || echo "AllowTcpForwarding yes" >> /etc/ssh/sshd_config'
  - 'rc-service sshd reload'
  - 'chown -R {{ .SSHUser }}:{{ .SSHUser }} /home/{{.SSHUser}}'
  - 'chmod 755 /home/{{ .SSHUser }}'
  - 'chmod 700 /home/{{ .SSHUser }}/.ssh'
  - 'addgroup {{.SSHUser}} docker'
  - sed -i 's/timeout=10/timeout=0/g' /boot/grub/grub.cfg
  - sed -i 's/GRUB_TIMEOUT=10/GRUB_TIMEOUT=0/g' /etc/default/grub
`
