package builtin

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strings"

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
	return &CalculatorTool{BaseTool: base}
}

func (t *CalculatorTool) Run(parameters map[string]any) tools.ToolResponse {
	expression, _ := parameters["input"].(string)
	if expression == "" {
		expression, _ = parameters["expression"].(string)
	}
	if expression == "" {
		return tools.Error("计算表达式不能为空", tools.ToolErrorCodeInvalidParam, nil)
	}

	node, err := parser.ParseExpr(strings.TrimSpace(expression))
	if err != nil {
		return tools.Error(fmt.Sprintf("表达式语法错误: %v", err), tools.ToolErrorCodeInvalidFormat, map[string]any{
			"expression": expression,
		})
	}

	val, err := evalNode(node)
	if err != nil {
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

func Calculate(expression string) string {
	tool := NewCalculatorTool()
	resp := tool.Run(map[string]any{"input": expression})
	return resp.Text
}

func evalExpression(expr string) (float64, error) {
	node, err := parser.ParseExpr(strings.TrimSpace(expr))
	if err != nil {
		return 0, err
	}
	return evalNode(node)
}

func evalNode(node ast.Expr) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		var v float64
		_, err := fmt.Sscanf(n.Value, "%f", &v)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", n.Value)
		}
		return v, nil
	case *ast.ParenExpr:
		return evalNode(n.X)
	case *ast.UnaryExpr:
		val, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.SUB:
			return -val, nil
		case token.ADD:
			return val, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator: %s", n.Op.String())
		}
	case *ast.CallExpr:
		funcIdent, ok := n.Fun.(*ast.Ident)
		if !ok {
			return 0, fmt.Errorf("unsupported function expression")
		}
		args := make([]float64, 0, len(n.Args))
		for _, arg := range n.Args {
			value, err := evalNode(arg)
			if err != nil {
				return 0, err
			}
			args = append(args, value)
		}
		return evalFunction(funcIdent.Name, args)
	case *ast.Ident:
		switch n.Name {
		case "pi":
			return math.Pi, nil
		case "e":
			return math.E, nil
		default:
			return 0, fmt.Errorf("undefined variable: %s", n.Name)
		}
	case *ast.BinaryExpr:
		left, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		right, err := evalNode(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		case token.XOR:
			return float64(int(left) ^ int(right)), nil
		default:
			return 0, fmt.Errorf("unsupported operator: %s", n.Op.String())
		}
	default:
		return 0, fmt.Errorf("unsupported expression type")
	}
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
		return 0, fmt.Errorf("unsupported function: %s", name)
	}
}

func formatNumber(v float64) string {
	if math.Trunc(v) == v {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%g", v)
}
