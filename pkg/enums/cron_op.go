//go:generate go run github.com/dmarkham/enumer -trimprefix=CronOp -type=CronOp -json -text

package enums

type CronOp int

const (
	CronOpNew    CronOp = iota // new scheduled function
	CronOpUpdate               // function config updated
	CronOpPause
	CronOpUnpause // function unpaused, resume crons.
	CronOpProcess
	CronInit // function enrolled in system queue for crons
)
