package builtin

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"helloagents-go/hello_agents/tools"
)

type CalculatorTool struct {
	tools.BaseTool
}

func NewCalculatorTool() *CalculatorTool {
	base := tools.NewBaseTool("python_calculator", "执行数学计算。支持基本运算、数学函数等。例如：2+3*4, sqrt(16), sin(pi/2)等。", false)
	base.Parameters = map[string]tools.ToolParameter{
		"input": {
			Name:        "input",
			Type:        "string",
			Description: "要计算的数学表达式，支持基本运算和数学函数",
			Required:    true,
		},
	}
	t := &CalculatorTool{BaseTool: base}
	t.BaseTool.SetRunImpl(t.Run)
	return t
}

func (t *CalculatorTool) Run(parameters map[string]any) tools.ToolResponse {
	expression, _ := parameters["input"].(string)
	if expression == "" {
		expression, _ = parameters["expression"].(string)
	}
	if expression == "" {
		return tools.Error("计算表达式不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}

	val, err := evalExpression(strings.TrimSpace(expression))
	if err != nil {
		var syntaxErr calcSyntaxError
		if errors.As(err, &syntaxErr) {
			return tools.Error(fmt.Sprintf("表达式语法错误: %v", err), tools.ToolErrorCodeInvalidFormat, map[string]any{
				"expression": expression,
			})
		}
		return tools.Error(fmt.Sprintf("计算失败: %v", err), tools.ToolErrorCodeExecutionError, map[string]any{
			"expression": expression,
		})
	}

	resultStr := formatNumber(val)
	return tools.Success(
		fmt.Sprintf("计算结果: %s", resultStr),
		map[string]any{
			"expression":  expression,
			"result":      val,
			"result_str":  resultStr,
			"result_type": "float64",
		},
		nil,
	)
}

func Calculate(expression string) tools.ToolResponse {
	tool := NewCalculatorTool()
	return tool.Run(map[string]any{"input": expression})
}

type calcSyntaxError struct {
	message string
}

func (e calcSyntaxError) Error() string {
	return e.message
}

type calcTokenType int

const (
	calcTokenEOF calcTokenType = iota
	calcTokenNumber
	calcTokenIdent
	calcTokenPlus
	calcTokenMinus
	calcTokenMul
	calcTokenDiv
	calcTokenPow
	calcTokenBitXor
	calcTokenLParen
	calcTokenRParen
	calcTokenComma
)

type calcToken struct {
	typ   calcTokenType
	text  string
	value float64
}

func evalExpression(expr string) (float64, error) {
	tokens, err := tokenizeExpression(expr)
	if err != nil {
		return 0, err
	}
	parser := &calcParser{
		tokens: tokens,
		pos:    0,
	}
	return parser.parse()
}

func tokenizeExpression(input string) ([]calcToken, error) {
	runes := []rune(input)
	tokens := make([]calcToken, 0, len(runes))

	for i := 0; i < len(runes); {
		ch := runes[i]
		if unicode.IsSpace(ch) {
			i++
			continue
		}

		if unicode.IsDigit(ch) || ch == '.' {
			start := i
			literal, next, err := readNumberLiteral(runes, i)
			if err != nil {
				return nil, err
			}
			value, err := strconv.ParseFloat(literal, 64)
			if err != nil {
				return nil, calcSyntaxError{message: fmt.Sprintf("无效数字: %s", literal)}
			}
			tokens = append(tokens, calcToken{typ: calcTokenNumber, text: string(runes[start:next]), value: value})
			i = next
			continue
		}

		if unicode.IsLetter(ch) || ch == '_' {
			start := i
			i++
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			tokens = append(tokens, calcToken{
				typ:  calcTokenIdent,
				text: string(runes[start:i]),
			})
			continue
		}

		switch ch {
		case '+':
			tokens = append(tokens, calcToken{typ: calcTokenPlus, text: "+"})
			i++
		case '-':
			tokens = append(tokens, calcToken{typ: calcTokenMinus, text: "-"})
			i++
		case '/':
			tokens = append(tokens, calcToken{typ: calcTokenDiv, text: "/"})
			i++
		case '^':
			tokens = append(tokens, calcToken{typ: calcTokenBitXor, text: "^"})
			i++
		case '(':
			tokens = append(tokens, calcToken{typ: calcTokenLParen, text: "("})
			i++
		case ')':
			tokens = append(tokens, calcToken{typ: calcTokenRParen, text: ")"})
			i++
		case ',':
			tokens = append(tokens, calcToken{typ: calcTokenComma, text: ","})
			i++
		case '*':
			if i+1 < len(runes) && runes[i+1] == '*' {
				tokens = append(tokens, calcToken{typ: calcTokenPow, text: "**"})
				i += 2
			} else {
				tokens = append(tokens, calcToken{typ: calcTokenMul, text: "*"})
				i++
			}
		default:
			return nil, calcSyntaxError{message: fmt.Sprintf("不支持的字符: %q", ch)}
		}
	}

	tokens = append(tokens, calcToken{typ: calcTokenEOF, text: ""})
	return tokens, nil
}

func readNumberLiteral(runes []rune, start int) (string, int, error) {
	i := start
	sawDigit := false
	sawDot := false

	for i < len(runes) {
		ch := runes[i]
		if unicode.IsDigit(ch) {
			sawDigit = true
			i++
			continue
		}
		if ch == '.' && !sawDot {
			sawDot = true
			i++
			continue
		}
		break
	}

	if !sawDigit {
		return "", start, calcSyntaxError{message: "无效数字格式"}
	}

	if i < len(runes) && (runes[i] == 'e' || runes[i] == 'E') {
		i++
		if i < len(runes) && (runes[i] == '+' || runes[i] == '-') {
			i++
		}
		expStart := i
		for i < len(runes) && unicode.IsDigit(runes[i]) {
			i++
		}
		if expStart == i {
			return "", start, calcSyntaxError{message: "科学计数法指数部分缺失"}
		}
	}

	return string(runes[start:i]), i, nil
}

type calcParser struct {
	tokens []calcToken
	pos    int
}

func (p *calcParser) parse() (float64, error) {
	value, err := p.parseBitXor()
	if err != nil {
		return 0, err
	}
	if p.current().typ != calcTokenEOF {
		return 0, calcSyntaxError{message: fmt.Sprintf("意外的 token: %s", p.current().text)}
	}
	return value, nil
}

func (p *calcParser) parseBitXor() (float64, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return 0, err
	}

	for p.match(calcTokenBitXor) {
		right, err := p.parseAddSub()
		if err != nil {
			return 0, err
		}
		left = float64(int(left) ^ int(right))
	}

	return left, nil
}

func (p *calcParser) parseAddSub() (float64, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return 0, err
	}

	for {
		if p.match(calcTokenPlus) {
			right, err := p.parseMulDiv()
			if err != nil {
				return 0, err
			}
			left += right
			continue
		}
		if p.match(calcTokenMinus) {
			right, err := p.parseMulDiv()
			if err != nil {
				return 0, err
			}
			left -= right
			continue
		}
		break
	}

	return left, nil
}

func (p *calcParser) parseMulDiv() (float64, error) {
	left, err := p.parseFactor()
	if err != nil {
		return 0, err
	}

	for {
		if p.match(calcTokenMul) {
			right, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			left *= right
			continue
		}
		if p.match(calcTokenDiv) {
			right, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
			continue
		}
		break
	}

	return left, nil
}

func (p *calcParser) parseFactor() (float64, error) {
	if p.match(calcTokenPlus) {
		return p.parseFactor()
	}
	if p.match(calcTokenMinus) {
		value, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		return -value, nil
	}
	return p.parsePower()
}

func (p *calcParser) parsePower() (float64, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return 0, err
	}
	if p.match(calcTokenPow) {
		right, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		left = math.Pow(left, right)
	}
	return left, nil
}

func (p *calcParser) parsePrimary() (float64, error) {
	token := p.current()
	switch token.typ {
	case calcTokenNumber:
		p.pos++
		return token.value, nil
	case calcTokenIdent:
		p.pos++
		if p.match(calcTokenLParen) {
			args := []float64{}
			if p.current().typ != calcTokenRParen {
				for {
					arg, err := p.parseBitXor()
					if err != nil {
						return 0, err
					}
					args = append(args, arg)
					if p.match(calcTokenComma) {
						continue
					}
					break
				}
			}
			if !p.match(calcTokenRParen) {
				return 0, calcSyntaxError{message: "缺少右括号 ')'"}
			}
			return evalFunction(token.text, args)
		}
		switch token.text {
		case "pi":
			return math.Pi, nil
		case "e":
			return math.E, nil
		default:
			return 0, fmt.Errorf("未定义的变量: %s", token.text)
		}
	case calcTokenLParen:
		p.pos++
		value, err := p.parseBitXor()
		if err != nil {
			return 0, err
		}
		if !p.match(calcTokenRParen) {
			return 0, calcSyntaxError{message: "缺少右括号 ')'"}
		}
		return value, nil
	default:
		if token.typ == calcTokenEOF {
			return 0, calcSyntaxError{message: "表达式不完整"}
		}
		return 0, calcSyntaxError{message: fmt.Sprintf("意外的 token: %s", token.text)}
	}
}

func (p *calcParser) current() calcToken {
	if p.pos >= len(p.tokens) {
		return calcToken{typ: calcTokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *calcParser) match(tokenType calcTokenType) bool {
	if p.current().typ == tokenType {
		p.pos++
		return true
	}
	return false
}

func evalFunction(name string, args []float64) (float64, error) {
	switch name {
	case "abs":
		if len(args) != 1 {
			return 0, fmt.Errorf("abs requires 1 argument")
		}
		return math.Abs(args[0]), nil
	case "round":
		if len(args) == 1 {
			return math.Round(args[0]), nil
		}
		if len(args) == 2 {
			p := math.Pow10(int(args[1]))
			return math.Round(args[0]*p) / p, nil
		}
		return 0, fmt.Errorf("round requires 1 or 2 arguments")
	case "max":
		if len(args) == 0 {
			return 0, fmt.Errorf("max requires at least 1 argument")
		}
		v := args[0]
		for _, item := range args[1:] {
			if item > v {
				v = item
			}
		}
		return v, nil
	case "min":
		if len(args) == 0 {
			return 0, fmt.Errorf("min requires at least 1 argument")
		}
		v := args[0]
		for _, item := range args[1:] {
			if item < v {
				v = item
			}
		}
		return v, nil
	case "sum":
		total := 0.0
		for _, item := range args {
			total += item
		}
		return total, nil
	case "sqrt":
		if len(args) != 1 {
			return 0, fmt.Errorf("sqrt requires 1 argument")
		}
		return math.Sqrt(args[0]), nil
	case "sin":
		if len(args) != 1 {
			return 0, fmt.Errorf("sin requires 1 argument")
		}
		return math.Sin(args[0]), nil
	case "cos":
		if len(args) != 1 {
			return 0, fmt.Errorf("cos requires 1 argument")
		}
		return math.Cos(args[0]), nil
	case "tan":
		if len(args) != 1 {
			return 0, fmt.Errorf("tan requires 1 argument")
		}
		return math.Tan(args[0]), nil
	case "log":
		if len(args) == 1 {
			return math.Log(args[0]), nil
		}
		if len(args) == 2 {
			return math.Log(args[0]) / math.Log(args[1]), nil
		}
		return 0, fmt.Errorf("log requires 1 or 2 arguments")
	case "exp":
		if len(args) != 1 {
			return 0, fmt.Errorf("exp requires 1 argument")
		}
		return math.Exp(args[0]), nil
	default:
		return 0, fmt.Errorf("不支持的函数: %s", name)
	}
}

func formatNumber(v float64) string {
	if math.Trunc(v) == v {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%g", v)
}
