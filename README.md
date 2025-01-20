# Restic Wrapper

`restic_wrapper` is a Go-based tool that extends the functionality of the [restic](https://restic.net) backup tool. It
offers additional features and configuration options for backing up your macOS system.

## Features

- **AWS CloudWatch Integration**: This program sends backup metrics to AWS CloudWatch, allowing you to monitor your
  backups and set up alerts based on these metrics. This helps you track backup status and receive timely notifications
  in case of issues.
- **Concurrent Backup Prevention**: Restic Wrapper prevents multiple instances of the same backup from running
  simultaneously. It uses a lock file to ensure that only one backup process is active at a time, avoiding conflicts and
  resource contention.
- **Secure Sensitive Data**: The program retrieves sensitive information, such as AWS credentials and restic repository
  details, from the macOS Keychain.
- **AC Power Requirement**: Restic Wrapper can be configured to run backups only when your laptop is connected to AC
  power, ensuring that backups are performed when the system is stable and not relying on battery power.
- **macOS Compatibility**: Restic Wrapper is designed to work exclusively on macOS, leveraging macOS-specific commands
  and features for backup operations.

## Requirements

- macOS
- Go 1.23.4 or later
- `restic` installed and accessible in the system path
- AWS credentials configured in the macOS keychain

## Installation

1. Clone the repository:
   ```sh
   git clone https://github.com/myarik/restic_wrapper.git
   cd restic_wrapper
   ```

2. Build the project:
   ```sh
   make build
   ```

## Configuration

Create a configuration file named `config.yaml` in the `~/.restic_backup` directory with the following structure:

```yaml
backup_directory: "/path/to/backup"
lock_file: ".restic_backup_lock"
log_file: "restic_backup.log"

restic:
  executable_path: "/usr/local/bin/restic"
  files_from: "backup.txt"
  exclude_file: "exclude.txt"
  s3_storage_class: "STANDARD_IA"

host_name: "your-hostname"
security_service: "restic_backup"
require_ac_power: true
cleanup_old_backups: false
```

- `backup_directory`: Directory for backup-related files and logs.
- `lock_file`:  The lock file to prevent concurrent backups.
- `log_file`: The log file.
- `restic.executable_path`: Path to the restic executable.
- `restic.files_from`: The file containing the list of files and directories to back up.
- `restic.exclude_file`: The file containing the list of files and directories to exclude from the backup.
- `restic.s3_storage_class`: S3 storage class for the backup.
- `host_name`: Hostname of the system.
- `security_service`: The macOS Keychain service for storing sensitive data.
- `require_ac_power`: Boolean indicating whether to require AC power for running backups.
- `cleanup_old_backups`: Boolean indicating whether to clean up old backups.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## Acknowledgements

- [restic](https://restic.net/)
- [AWS SDK for Go](https://aws.amazon.com/sdk-for-go/)
- [Logrus](https://github.com/sirupsen/logrus)
- [Viper](https://github.com/spf13/viper)
- [lumberjack](https://github.com/natefinch/lumberjack)

## Contact

For any questions or support, please open an issue on the GitHub repository.