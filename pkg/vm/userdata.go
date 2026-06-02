package vm

const userDataTemplate = `#cloud-config
---
ssh_pwauth: true
hostname: {{.ProfileName}}
chpasswd:
  expire: false
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
  - path: /etc/docker/daemon.json
    content: |
      {
        "storage-driver": "fuse-overlayfs",
        "hosts": ["unix:///var/run/docker.sock"]
      }

# package_update: true
# package_upgrade: true
# package_reboot_if_required: true
# # package_reboot: true
packages:
  - docker
  - socat
  - mount
  - fuse-overlayfs
  - linux-virt

runcmd:
  - |
    # Enable parallel service booting in OpenRC
    # sed -i 's/#rc_parallel="NO"/rc_parallel="YES"/g' /etc/rc.conf

    # Register OpenRC boot processes
    rc-update add cgroups default
    rc-update add docker default

    # Fire up components in the correct sequence
    rc-service cgroups start
    rc-service docker start

    # routing forward tables
    echo 1 > /proc/sys/net/ipv4/ip_forward

    # Enable SSH TCP forwarding
    sed -i 's/AllowTcpForwarding no/AllowTcpForwarding yes/g' /etc/ssh/sshd_config
    grep -q "^AllowTcpForwarding yes" /etc/ssh/sshd_config || echo "AllowTcpForwarding yes" >> /etc/ssh/sshd_config
    rc-service sshd reload

    chown -R {{ .SSHUser }}:{{ .SSHUser }} /home/{{.SSHUser}}
    chmod 755 /home/{{ .SSHUser }}
    chmod 700 /home/{{ .SSHUser }}/.ssh
`
