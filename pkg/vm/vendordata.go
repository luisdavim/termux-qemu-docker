package vm

const vendorDataTemplate = `#cloud-config
---
system_info:
  default_user:
    name: alpine

# Disable blocking network detection
manage_etc_hosts: false

cloud_final_modules:
  - package_update_upgrade_install
  - write_files_deferred
  - scripts_vendor
  - scripts_per_once
  - scripts_per_boot
  - scripts_per_instance
  - scripts_user
  - ssh_authkey_fingerprints
  - keys_to_console
  - final_message
  - power_state_change
`
