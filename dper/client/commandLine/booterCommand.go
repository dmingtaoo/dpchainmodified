package commandLine

import (
	"dpchain/dper"
	"dpchain/dper/configer"
	"dpchain/utils"
	"fmt"
	"os"
	"strings"
	"time"
)

var sleepTime time.Duration = 3 * time.Second
var sleepLongTime time.Duration = 5 * time.Second

func (c *CommandLine) runBooterCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
	if len(arrCommandStr) == 0 {
		return nil
	}
	switch arrCommandStr[0] {
	case "exit", "Exit", "EXIT", "quit", "Quit", "QUIT":
		c.printByeBye()
		os.Exit(0)
	case "init", "Init", "INIT":
		fmt.Println("Loading config...")
		booterCfg, err := configer.LoadToBooterCfg(defaultBooterConfigFilePath)
		if err != nil {
			return err
		}
		fmt.Println("Config load succeed.")
		oss, err := dper.NewOrderServiceServer(booterCfg)
		if err != nil {
			return err
		}
		c.orderServiceServer = oss
		fmt.Println("Booter has initialized!")
	case "stateSelf", "state":
		if c.orderServiceServer == nil {
			return fmt.Errorf("should first use command \"init\" to initialize the booter!")
		}
		err := c.orderServiceServer.StateSelf()
		if err != nil {
			return err
		}
		fmt.Println("State self succeed.")
	case "construct", "constructDpnet":
		if c.orderServiceServer == nil {
			return fmt.Errorf("should first use command \"init\" to initialize the booter!")
		}
		err := c.orderServiceServer.ConstructDpnet()
		if err != nil {
			return err
		}
		fmt.Println("Construct dpnet succeed.")
	case "start", "Start", "START":
		if c.orderServiceServer == nil {
			return fmt.Errorf("should first use command \"init\" to initialize the booter!")
		}
		err := c.orderServiceServer.Start()
		if err != nil {
			return err
		}
		c.running = true
		fmt.Println("Booter starts and succeeded in joining DP-Chain!")
	case "saveurl":
		url, err := c.orderServiceServer.BackSelfUrl()
		if err != nil {
			return err
		}
		fmt.Printf("The booter self url is: %s\n", url)
		err = utils.SaveString(url, defaultBooterUrlSavePath)
		if err != nil {
			return err
		}
		fmt.Println("Url save: succeed.")
	case "help":
		c.printUsages()
	case "viewNet":
		if c.orderServiceServer == nil {
			return fmt.Errorf("should first use command \"init\" to initialize the booter!")
		}
		netInfo := c.orderServiceServer.BackViewNetInfo()
		fmt.Print(netInfo)
	case "sleep":
		time.Sleep(sleepTime)
	case "sleeplong":
		time.Sleep(sleepLongTime)
	default:
		fmt.Println("Unknown command is input. Pint \"help\" to see all commands")

	}
	return nil

}
