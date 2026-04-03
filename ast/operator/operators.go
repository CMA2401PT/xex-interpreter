package operator

type Operator byte

const (
	OperatorILLEGAL = Operator(iota)
	ASSIGN
	PLUS
	MINUS
	BANG
	ASTERISK
	SLASH
	FLOOR_DIV
	PERCENT
	LT
	GT
	EQ
	NOT_EQ
)

func (o Operator) String() string {
	return OperatorToString[o]
}

var StringToOperator = map[string]Operator{
	"=":  ASSIGN,
	"+":  PLUS,
	"-":  MINUS,
	"!":  BANG,
	"*":  ASTERISK,
	"/":  SLASH,
	"//": FLOOR_DIV,
	"%":  PERCENT,
	"<":  LT,
	">":  GT,
	"==": EQ,
	"!=": NOT_EQ,
}

var OperatorToString = map[Operator]string{
	OperatorILLEGAL: "ILLEGAL",
	ASSIGN:          "=",
	PLUS:            "+",
	MINUS:           "-",
	BANG:            "!",
	ASTERISK:        "*",
	SLASH:           "/",
	FLOOR_DIV:       "//",
	PERCENT:         "%",
	LT:              "<",
	GT:              ">",
	EQ:              "==",
	NOT_EQ:          "!=",
}
