package vm

// tiny-cloud doesn't seem to support write_files or - | in runcmd
// this is formatted to support both tiny-cloud and cloud-init
// tiny-cloud also doesn't add non existing groups so we add docker explicity at the end
const userDataTemplate = `#cloud-config
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
  - 'mkdir -p /etc/doas.d'
  - 'echo "permit nopass {{ .SSHUser }} as root" > /etc/doas.d/custom.conf'
  - 'mkdir -p /etc/docker'
  - 'echo "{\"storage-driver\": \"fuse-overlayfs\", \"hosts\": [\"unix:///var/run/docker.sock\"]}" > /etc/docker/daemon.json'
  - 'rc-update add cgroups default'
  - 'rc-update add docker default'
  - 'rc-service cgroups start'
  - 'rc-service docker start'
  - 'echo 1 > /proc/sys/net/ipv4/ip_forward'
  - 'sed -i "s/#PasswordAuthentication yes/PasswordAuthentication yes/g" /etc/ssh/sshd_config'
  - 'sed -i "s/PasswordAuthentication no/PasswordAuthentication yes/g" /etc/ssh/sshd_config'
  - 'sed -i "s/AllowTcpForwarding no/AllowTcpForwarding yes/g" /etc/ssh/sshd_config'
  - 'grep -q "^AllowTcpForwarding yes" /etc/ssh/sshd_config || echo "AllowTcpForwarding yes" >> /etc/ssh/sshd_config'
  - 'rc-service sshd reload'
  - 'chown -R {{ .SSHUser }}:{{ .SSHUser }} /home/{{.SSHUser}}'
  - 'chmod 755 /home/{{ .SSHUser }}'
  - 'chmod 700 /home/{{ .SSHUser }}/.ssh'
  - 'addgroup {{.SSHUser}} docker'
`
