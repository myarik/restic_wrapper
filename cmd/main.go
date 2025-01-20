package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/gofrs/flock"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Config represents the configuration for the program
type Config struct {
	BackupDir string `mapstructure:"backup_directory"`
	LockFile  string `mapstructure:"lock_file"`
	LogFile   string `mapstructure:"log_file"`

	Restic struct {
		Path        string `mapstructure:"executable_path"`
		FilesFrom   string `mapstructure:"files_from"`
		ExcludeFile string `mapstructure:"exclude_file"`
		S3Storage   string `mapstructure:"s3_storage_class"`
	} `mapstructure:"restic"`

	HostName          string `mapstructure:"host_name"`
	SecurityService   string `mapstructure:"security_service"`
	RequireAcPower    bool   `mapstructure:"require_ac_power"`
	CleanupOldBackups bool   `mapstructure:"cleanup_old_backups"`
}

var (
	appConfig Config
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error getting user home directory:", err)
	}
	/// Set default values for configuration variables
	viper.SetDefault("backup_directory", filepath.Join(homeDir, ".restic_backup"))
	viper.SetDefault("lock_file", ".restic_backup_lock")
	viper.SetDefault("log_file", "restic_backup.log")

	viper.SetDefault("restic.executable_path", "/usr/local/bina/restic")
	viper.SetDefault("restic.files_from", "backup.txt")
	viper.SetDefault("restic.exclude_file", "exclude.txt")
	viper.SetDefault("restic.s3_storage_class", "STANDARD_IA")

	viper.SetDefault("host_name", "localhost")
	viper.SetDefault("security_service", "restic_backup")

	viper.SetDefault("require_ac_power", true)
	viper.SetDefault("cleanup_old_backups", false)

	// Read the configuration from the config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Join(homeDir, ".restic_backup"))

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	if err := viper.Unmarshal(&appConfig); err != nil {
		log.Fatalf("Error unmarshaling config: %v", err)
	}
	setupLogging()
}

// setupLogging configures the logrus logger
func setupLogging() {
	log.SetOutput(&lumberjack.Logger{
		Filename:   filepath.Join(appConfig.BackupDir, "logs", appConfig.LogFile),
		MaxSize:    10, // megabytes
		MaxBackups: 3,  // number of backups
		MaxAge:     28, // days
		LocalTime:  true,
	})
	log.SetLevel(log.InfoLevel)
}

// isOnPower checks if the system is running on AC power
func isOnPower() (bool, error) {
	cmd := exec.CommandContext(context.TODO(), "pmset", "-g", "ps", "|", "grep", "head", "-1")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to execute pmset: %w", err)
	}

	outputStr := strings.TrimSpace(string(output))
	return strings.Contains(outputStr, "AC Power"), nil
}

// getSecurityData retrieves the password for the given service from the macOS keychain
func getSecurityData(service, account string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel() // The cancel should be deferred so resources are cleaned up

	// Prepare the command and its arguments
	cmd := exec.CommandContext(ctx, "security", []string{"find-generic-password", "-s", service, "-a", account, "-w"}...)
	// Capture the output
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.WithFields(log.Fields{
			"cmd": "security",
			"err": err,
			"srv": service,
		}).Fatal("cannot get security data")
	}
	// Return the password output, trimming any trailing newline
	return string(bytes.TrimSpace(out))
}

// runResticCommand runs the restic command with the given arguments
func runResticCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, appConfig.Restic.Path, args...)
	cmd.Env = os.Environ()

	// Capture the command's stdout and stderr
	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
			if line != "" {
				log.WithFields(log.Fields{
					"cmd":       appConfig.Restic.Path,
					"operation": args[0],
				}).Error(line)
			}
		}
		log.WithFields(log.Fields{
			"cmd":       appConfig.Restic.Path,
			"operation": args[0],
			"err":       err,
		}).Error("failed to execute the command")
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line != "" {
			log.WithFields(log.Fields{
				"cmd":       appConfig.Restic.Path,
				"operation": args[0],
			}).Info(line)
		}
	}
	return nil
}

// sendAwsMetrics sends the backup metrics to AWS CloudWatch
func sendAwsMetrics(ctx context.Context, duration time.Duration) error {
	// Load the SDK's configuration from environment and shared config, and create a new client
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.WithField("err", err).Error("cannot load AWS SDK config")
		return err
	}

	// Create a new CloudWatch client
	svc := cloudwatch.NewFromConfig(cfg)

	// Create the input for the PutMetricData operation
	input := &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("ResticBackup"),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String("BackupDuration"),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("Environment"),
						Value: aws.String(appConfig.HostName),
					},
				},
				Timestamp: aws.Time(time.Now()),
				Unit:      types.StandardUnitSeconds,
				Value:     aws.Float64(duration.Seconds()),
			},
			{
				MetricName: aws.String("BackupCount"),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("Environment"),
						Value: aws.String(appConfig.HostName),
					},
				},
				Timestamp: aws.Time(time.Now()),
				Unit:      types.StandardUnitCount,
				Value:     aws.Float64(1),
			},
		},
	}

	// Send the metric data to CloudWatch
	_, err = svc.PutMetricData(ctx, input)
	if err != nil {
		log.WithField("err", err).Error("cannot put metric data to CloudWatch")
		return err
	}
	log.Info("Sent backup metrics to CloudWatch")
	return nil
}

// setupEnv sets up the environment variables for the restic command
func setupEnv() {
	os.Setenv("AWS_DEFAULT_REGION", getSecurityData(appConfig.SecurityService, "aws-region"))
	os.Setenv("AWS_ACCESS_KEY_ID", getSecurityData(appConfig.SecurityService, "aws-access-key-id"))
	os.Setenv("AWS_SECRET_ACCESS_KEY", getSecurityData(appConfig.SecurityService, "aws-secret-access-key"))
	os.Setenv("RESTIC_REPOSITORY", getSecurityData(appConfig.SecurityService, "repository"))
	os.Setenv("RESTIC_PASSWORD", getSecurityData(appConfig.SecurityService, "password"))
}

func main() {
	startTime := time.Now()

	fileLock := flock.New(filepath.Join(appConfig.BackupDir, appConfig.LockFile))
	locked, err := fileLock.TryLock()
	if err != nil {
		log.WithField("err", err).Error("cannot lock the lock file")
		os.Exit(1)
	}
	if !locked {
		log.Warn("Another instance of the program is already running. Exiting.")
		return
	}
	defer fileLock.Unlock()

	// Create a new context and add a timeout to it
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel() // The cancel should be deferred so resources are cleaned up

	if _, err = exec.LookPath(appConfig.Restic.Path); err != nil {
		log.WithField("cmd", appConfig.Restic.Path).Error("cannot find the restic command")
		return
	}

	// Check if the system is running on AC power
	isAcPower, err := isOnPower()
	if err != nil {
		log.WithField("err", err).Error("cannot check if the system is running on AC power")
		return
	}
	if appConfig.RequireAcPower && !isAcPower {
		log.Warn("The system is not running on AC power. Skipping backup.")
		return
	}

	setupEnv()

	if err = runResticCommand(ctx, "backup",
		"-o", "s3.storage-class="+appConfig.Restic.S3Storage,
		"--files-from", filepath.Join(appConfig.BackupDir, appConfig.Restic.FilesFrom),
		"--exclude-file", filepath.Join(appConfig.BackupDir, appConfig.Restic.ExcludeFile),
	); err != nil {
		log.WithFields(log.Fields{
			"cmd":     appConfig.Restic.Path,
			"command": "backup",
		}).Errorf("Backup failed")
		os.Exit(1)
	}
	if appConfig.CleanupOldBackups {
		if err = runResticCommand(ctx, "forget", "-q",
			"--prune",
			"--keep-hourly", "4",
			"--keep-daily", "7",
			"--keep-weekly", "5",
			"--keep-monthly", "12",
			"--keep-yearly", "5",
			"--keep-tag", "nodelete",
		); err != nil {
			log.WithFields(log.Fields{
				"cmd":     appConfig.Restic.Path,
				"command": "forget",
			}).Errorf("Forget failed")
		}
	}
	elapsedTime := time.Since(startTime)
	if err = sendAwsMetrics(ctx, elapsedTime); err != nil {
		log.WithField("err", err).Error("cannot send backup metrics to CloudWatch")
	}
	log.WithFields(log.Fields{
		"duration": elapsedTime,
	}).Info("Backup completed successfully")
}
