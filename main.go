package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFile, databaseURL string
var debug bool
var step int

type Contract struct {
	Hash                 string
	CreatorAddress       string
	ContractAddress      string
	CreationCode         string
	SourceCode           string
	ContractFile         string
	ContractName         string
	CompilerVersion      string
	CompilerType         string
	SolcVersion          string
	EvmVersion           string
	Optimization         string
	Runs                 string
	ABI                  string
	ConstructorArguments string
}

func main() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (default is config/local.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debugging(default false)")
	rootCmd.PersistentFlags().StringVar(&databaseURL, "database", "", "database url (default postgres://aurora:aurora@database/aurora)")
	cobra.CheckErr(rootCmd.Execute())
}

func initConfig() {
	if configFile != "" {
		log.Warn().Msg(fmt.Sprint("Using config file:", viper.ConfigFileUsed()))
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath("config")
		viper.AddConfigPath("../../config")
		viper.SetConfigName("local")
		if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
			panic(fmt.Errorf("Flags are not bindable: %v\n", err))
		}
	}
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err == nil {
		log.Warn().Msg(fmt.Sprint("Using config file:", viper.ConfigFileUsed()))
	}

	debug = viper.GetBool("debug")
	databaseURL = viper.GetString("database")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	step = 10000
}

var rootCmd = &cobra.Command{
	Use:     "indexer",
	Short:   "Imports verified contracts info to blockscout from aurorascan.",
	Long:    "Imports verified contracts info to blockscout from aurorascan.",
	Version: "0.0.1",
	Run: func(cmd *cobra.Command, args []string) {
		pgpool, err := pgxpool.Connect(context.Background(), databaseURL)
		if err != nil {
			panic(fmt.Errorf("unable to connect to database %s: %v", databaseURL, err))
		}
		defer pgpool.Close()

		i := 0
		err, head := getHead(pgpool)
		if err != nil {
			panic(err)
		}

		for {
			err, gapCount := getGapCount(pgpool, i)
			if err != nil {
				panic(err)
			}
			i++
			if gapCount < step {
				fmt.Printf("GapCount(%v:%v) - %v\n", step*i, step*(i+1), gapCount)
			}
			if i*10000 > head {
				os.Exit(0)
			}
		}

		interrupt := make(chan os.Signal, 10)
		signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGINT)

		select {
		case <-interrupt:
			os.Exit(0)
		}

	},
}

func getGapCount(pgpool *pgxpool.Pool, page int) (error, int) {
	selectSql := fmt.Sprintf("SELECT COUNT(1) FROM blocks WHERE blocks.number >= %v AND blocks.number < %v", step*page, step*(page+1))
	var gapCount int
	err := pgpool.QueryRow(context.Background(), selectSql).Scan(&gapCount)
	return err, gapCount
}

func getHead(pgpool *pgxpool.Pool) (error, int) {
	var block int
	err := pgpool.QueryRow(context.Background(), "SELECT MAX(blocks.number) FROM blocks LIMIT 1").Scan(&block)
	return err, block
}
