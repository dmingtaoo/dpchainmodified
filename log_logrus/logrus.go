package loglogrus

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-ini/ini"
	"github.com/sirupsen/logrus"
)

// 颜色
const (
	red    = 31
	yellow = 33
	blue   = 36
	gray   = 37
)

var (
	logLevel logrus.Level = logrus.WarnLevel // 默认等级
)

var Log *logrus.Logger

// 自定义Hook, 负责将warnf级别以上的日志单独放到一个日志文件中
type WarnHook struct {
	Writer io.Writer
}

// 将logrus.WarnLevel级别以上的日志写入到单独的文件中
func (hook *WarnHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}
	hook.Writer.Write([]byte(line))
	return nil
}

// 范围只限logrus.WarnLevel级别以上的日志
func (hook *WarnHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.WarnLevel, logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel}
}

// 将上下层共识消息单独写入到一个日志文件中
type ConsensusHook struct {
	tpbftWriter io.Writer
	raftWriter  io.Writer
}

func (hook *ConsensusHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}
	if strings.Contains(line, "[TPBFT Consensus]") {
		hook.tpbftWriter.Write([]byte(line))
	}
	if strings.Contains(line, "[RAFT Consensus]") {
		hook.raftWriter.Write([]byte(line))
	}

	return nil
}

// 不针对特定等级的日志消息
func (hook *ConsensusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

type LogFormatter struct{} //需要实现Formatter(entry *logrus.Entry) ([]byte, error)接口

func (t *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var levelColor int
	switch entry.Level { //根据不同的level去展示颜色
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = gray
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer //获取日志实例中的缓冲
	} else {
		b = &bytes.Buffer{}
	}
	//自定义日期格式
	timestamp := entry.Time.Format("2006-01-02 15:04:05") //获取格式化的日志消息时间
	if entry.HasCaller() {                                //检测日志实例是否开启  SetReportCaller(true)
		//自定义文件路径
		funcVal := entry.Caller.Function                                                 //生成日志消息的函数名
		fileVal := fmt.Sprintf("%s:%d", path.Base(entry.Caller.File), entry.Caller.Line) //生成日志的文件和日志行号
		//自定义输出格式
		//fmt.Fprintf(b, "[%s] \x1b[%dm[%s]\x1b[0m %s %s %s\n", timestamp, levelColor, entry.Level, fileVal, funcVal, entry.Message) //entry.Message是日志消息内容
		_ = levelColor
		fmt.Fprintf(b, "[%s] [%s] %s %s %s\n", timestamp, entry.Level, fileVal, funcVal, entry.Message) //entry.Message是日志消息内容
	} else {
		//fmt.Fprintf(b, "[%s] \x1b[%dm[%s]\x1b[0m %s\n", timestamp, levelColor, entry.Level, entry.Message)
		fmt.Fprintf(b, "[%s] [%s] %s\n", timestamp, entry.Level, entry.Message)
	}
	return b.Bytes(), nil //返回格式化的日志缓冲
}

// 自定义的io.Writer对象，负责对日志文件进行分割
type logFileWriter struct {
	file    *os.File // 真正的日志文件的io.Writer
	logPath string   // 日志文件路径
	logFile string   // 日志文件路径+文件名

	logKind string // 普通日志 or 错误日志

	fileDate string //判断日期切换目录
}

// 自定义write方法
func (p *logFileWriter) Write(data []byte) (n int, err error) {
	if p == nil {
		return 0, errors.New("logFileWriter is nil")
	}
	if p.file == nil {
		return 0, errors.New("file not opened")
	}

	//判断日期是否变更，如果变更需要重新生成新的日志文件，达到分割效果
	fileDate := time.Now().Format("2006_01_02_15_04") // 按分钟进行分割
	if p.fileDate != fileDate {
		p.file.Close() //关闭旧的日志文件

		p.logFile = p.logPath + fmt.Sprintf("%s%s", p.logKind, time.Now().Format("2006_01_02_15_04"))

		err = os.MkdirAll(fmt.Sprintf("%s", p.logPath), os.ModePerm)
		if err != nil {
			return 0, err
		}
		filename := fmt.Sprintf("%s", p.logFile)

		p.file, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660) //创建新的日志文件
		if err != nil {
			return 0, err
		}
	}

	n, e := p.file.Write(data) //将日志写入日志文件
	return n, e

}
func init() {

	Log = NewLog()

}

func NewLog() *logrus.Logger {

	dir, _ := os.Getwd()

	cfg, err := ini.Load("./settings/extended.ini") // 优先加载当前路径下的extended.ini
	if err != nil {
		errStr := fmt.Sprintf("Fail to parse './settings/http.ini': %v\n", err)
		fmt.Printf(errStr)
		log := logrus.StandardLogger() // 没有配置文件的情况下默认使用 std 作为输出设备
		log.SetLevel(logrus.PanicLevel)
		return log
	} else {
		level := cfg.Section("Log").Key("LogLevel").String()
		fmt.Printf("当前工作路径为:%s\n", dir)
		fmt.Println("***************************读取的日志等级为:", level, "*************************************")

		switch level {
		case "Trace":
			logLevel = logrus.TraceLevel
		case "Debug":
			logLevel = logrus.DebugLevel
		case "Infof":
			logLevel = logrus.InfoLevel
		case "Warnf":
			logLevel = logrus.WarnLevel
		case "Errorf":
			logLevel = logrus.ErrorLevel
		case "Fatal":
			logLevel = logrus.FatalLevel
		case "Panic":
			logLevel = logrus.PanicLevel
		default:
			logLevel = logrus.InfoLevel
		}

		mLog := logrus.New() //新建一个实例
		fileDate := time.Now().Format("2006_01_02_15_04")
		rootPath, _ := os.Getwd()
		logPath := rootPath + string(os.PathSeparator) + "log" + string(os.PathSeparator)

		commonLogPath := logPath + "log" + string(os.PathSeparator)
		commonLogFile := commonLogPath + fmt.Sprintf("log%s", fileDate)
		file, err := os.OpenFile(commonLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			mLog.Infof("Create LogFile is failed, err:%v", err)
		}

		fileWriter := logFileWriter{file, commonLogPath, commonLogFile, "log", fileDate} //具有自定义的Writer方法

		writers := []io.Writer{
			&fileWriter,
			//os.Stdout,
		}
		//  同时写文件和屏幕
		fileAndStdoutWriter := io.MultiWriter(writers...) //io.MultiWriter可以包含多个Writer对象
		mLog.SetOutput(fileAndStdoutWriter)
		mLog.SetReportCaller(true)         //开启返回函数名和行号
		mLog.SetFormatter(&LogFormatter{}) //设置自己定义的Formatter
		mLog.SetLevel(logLevel)            //设置最低的Level

		// 错误日志单独存储的Hook
		errLogPath := logPath + "errLog" + string(os.PathSeparator)
		errLogFile := errLogPath + fmt.Sprintf("err.log%s", fileDate)
		errFile, _ := os.OpenFile(errLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		errFileWriter := logFileWriter{errFile, errLogPath, errLogFile, "err.log", fileDate} //具有自定义的Writer方法
		errhook := &WarnHook{Writer: &errFileWriter}
		mLog.AddHook(errhook)

		// 共识日志单独存储的Hook
		consensusLogPath := logPath + "consensusLog" + string(os.PathSeparator)

		tpbftLogFile := consensusLogPath + fmt.Sprintf("tpbft.log%s", fileDate)
		tpbftFile, _ := os.OpenFile(tpbftLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		tpbftFileWriter := logFileWriter{tpbftFile, consensusLogPath, tpbftLogFile, "tpbft.log", fileDate} //具有自定义的Writer方法

		raftLogFile := consensusLogPath + fmt.Sprintf("raft.log%s", fileDate)
		raftFile, _ := os.OpenFile(raftLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		raftFileWriter := logFileWriter{raftFile, consensusLogPath, raftLogFile, "raft.log", fileDate} //具有自定义的Writer方法

		consensushook := &ConsensusHook{tpbftWriter: &tpbftFileWriter, raftWriter: &raftFileWriter}
		mLog.AddHook(consensushook)

		return mLog
	}

}
