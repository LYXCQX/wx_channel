package cmd

import (
	"fmt"
	"os"
	"wx_channel/internal/app"
	"wx_channel/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	port    int
	dev     string
)

var rootCmd = &cobra.Command{
	Use:   "wx_channel",
	Short: "WeChat Channel Video Downloader",
	Long:  `A tool to download videos from WeChat Channels with auto-decryption and de-duplication.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		// 应用标志到配置
		if port != 0 {
			cfg.SetPort(port)
		}

		// 创建并运行应用
		application := app.NewApp(cfg)
		application.Run()
	},
}

func Execute() {
	// 允许在 Windows 上直接双击运行（禁用 Mousetrap 检测）
	cobra.MousetrapHelpText = ""

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 持久化标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.wx_channel/config.yaml)")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 0, "Proxy server network port")
	rootCmd.PersistentFlags().StringVarP(&dev, "dev", "d", "", "Proxy server network device")

	// 绑定标志到 viper
	_ = viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	_ = viper.BindPFlag("dev", rootCmd.PersistentFlags().Lookup("dev"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}
	// 配置加载由 config.Load() 在 rootCmd.Run 中完成
}
