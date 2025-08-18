//go:generate go run github.com/dmarkham/enumer -trimprefix=CronOp -type=CronOp -json -text

package enums

type CronOp int

const (
	CronOpNew CronOp = iota
	CronOpUpdate
	CronOpArchive
	CronOpPause
	CronOpUnpause
	CronOpProcess
	CronInit
)
