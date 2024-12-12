package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hm-edu/harica/client"
	"github.com/spf13/cobra"
)

var (
	domains           []string
	csr               string
	transactionType   string
	requesterEmail    string
	requesterPassword string
	requesterTOTPSeed string
	validatorEmail    string
	validatorPassword string
	validatorTOTPSeed string
	debug             bool
)

// genCertCmd represents the genCert command
var genCertCmd = &cobra.Command{
	Use: "gen-cert",
	Run: func(cmd *cobra.Command, args []string) {

		requester, err := client.NewClient(requesterEmail, requesterPassword, requesterTOTPSeed, client.WithDebug(debug))
		if err != nil {
			slog.Error("failed to create requester client", slog.Any("error", err))
			os.Exit(1)
		}
		validator, err := client.NewClient(validatorEmail, validatorPassword, validatorTOTPSeed, client.WithDebug(debug))
		if err != nil {
			slog.Error("failed to create validator client", slog.Any("error", err))
			os.Exit(1)
		}

		d, err := requester.CheckDomainNames(domains)
		if err != nil {
			slog.Error("failed to check domain names", slog.Any("error", err))
			os.Exit(1)
		}
		transaction, err := requester.RequestCertificate(d, csr, transactionType)
		if err != nil {
			slog.Error("failed to request certificate", slog.Any("error", err))
			os.Exit(1)
		}

		reviews, err := validator.GetPendingReviews()
		if err != nil {
			slog.Error("failed to get pending reviews", slog.Any("error", err))
			os.Exit(1)
		}

		for _, r := range reviews {
			if r.TransactionID == transaction.TransactionID {
				for _, s := range r.ReviewGetDTOs {
					err = validator.ApproveRequest(s.ReviewID, "Auto Approval", s.ReviewValue)
					if err != nil {
						slog.Error("failed to approve request", slog.Any("error", err))
						os.Exit(1)
					}
				}
			}
		}
		cert, err := requester.GetCertificate(transaction.TransactionID)
		if err != nil {
			slog.Error("failed to get certificate", slog.Any("error", err))
			os.Exit(1)
		}
		fmt.Print(cert.PemBundle)
	},
}

func init() {
	rootCmd.AddCommand(genCertCmd)
	genCertCmd.Flags().StringSliceVarP(&domains, "domains", "d", []string{}, "Domains to request certificate for")
	genCertCmd.Flags().StringVar(&csr, "csr", "", "CSR to request certificate with")
	genCertCmd.Flags().StringVarP(&transactionType, "transaction-type", "t", "DV", "Transaction type to request certificate with")
	genCertCmd.Flags().StringVar(&requesterEmail, "requester-email", "", "Email of requester")
	genCertCmd.Flags().StringVar(&requesterPassword, "requester-password", "", "Password of requester")
	genCertCmd.Flags().StringVar(&requesterTOTPSeed, "requester-totp-seed", "", "TOTP seed of requester")
	genCertCmd.Flags().StringVar(&validatorEmail, "validator-email", "", "Email of validator")
	genCertCmd.Flags().StringVar(&validatorPassword, "validator-password", "", "Password of validator")
	genCertCmd.Flags().StringVar(&validatorTOTPSeed, "validator-totp-seed", "", "TOTP seed of validator")
	genCertCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
	genCertCmd.MarkFlagRequired("domains")
	genCertCmd.MarkFlagRequired("csr")
	genCertCmd.MarkFlagRequired("requester-email")
	genCertCmd.MarkFlagRequired("requester-password")
	genCertCmd.MarkFlagRequired("requester-totp-seed")
	genCertCmd.MarkFlagRequired("validator-email")
	genCertCmd.MarkFlagRequired("validator-password")
	genCertCmd.MarkFlagRequired("validator-totp-seed")
}
