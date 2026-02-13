package builtin

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"helloagents-go/HelloAgents-go/tools"
)

// Calculator is a tool for evaluating mathematical expressions.
type Calculator struct {
	*tools.BaseTool
}

// NewCalculator creates a new Calculator tool.
func NewCalculator() *Calculator {
	params := []tools.ToolParameter{
		{
			Name:        "expression",
			Type:        "string",
			Description: "The mathematical expression to evaluate. Supports basic operations (+, -, *, /, ^, %) and functions (sin, cos, tan, sqrt, abs, log, exp, etc.). Example: '2 * (3 + 4)' or 'sin(pi/2)'",
			Required:    true,
		},
	}

	return &Calculator{
		BaseTool: tools.NewBaseTool(
			"calculator",
			"Evaluates mathematical expressions. Supports basic operations (+, -, *, /, ^, %), parentheses, and common math functions (sin, cos, tan, sqrt, abs, log, exp, min, max, etc.).",
			params,
		),
	}
}

// Run evaluates the mathematical expression.
func (c *Calculator) Run(parameters map[string]interface{}) (string, error) {
	expr, exists := parameters["expression"]
	if !exists {
		return "", fmt.Errorf("expression parameter is required")
	}

	exprStr, ok := expr.(string)
	if !ok {
		return "", fmt.Errorf("expression must be a string")
	}

	// Parse and evaluate the expression
	result, err := c.evaluate(exprStr)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return fmt.Sprintf("%g", result), nil
}

// evaluate parses and evaluates a mathematical expression.
// This is a simple recursive descent parser.
func (c *Calculator) evaluate(expr string) (float64, error) {
	// Tokenize
	tokens, err := tokenize(expr)
	if err != nil {
		return 0, err
	}

	// Parse expression
	parser := &expressionParser{tokens: tokens}
	result, err := parser.parseExpression()
	if err != nil {
		return 0, err
	}

	// Check if we consumed all tokens
	if parser.pos < len(parser.tokens) {
		return 0, fmt.Errorf("unexpected token at position %d: %s", parser.pos, parser.tokens[parser.pos])
	}

	return result, nil
}

// Token represents a lexical token.
type token struct {
	typ  tokenType
	value string
	pos   int
}

type tokenType int

const (
	tokenNumber tokenType = iota
	tokenOperator
	tokenIdentifier
	tokenLeftParen
	tokenRightParen
	tokenComma
)

// tokenize converts a string into tokens.
func tokenize(expr string) ([]token, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	tokens := make([]token, 0)
	i := 0

	for i < len(expr) {
		ch := expr[i]

		// Skip whitespace
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			i++
			continue
		}

		// Number
		if ch == '.' || (ch >= '0' && ch <= '9') {
			start := i
			hasDot := ch == '.'
			i++
			for i < len(expr) {
				ch := expr[i]
				if ch >= '0' && ch <= '9' {
					i++
				} else if ch == '.' && !hasDot {
					hasDot = true
					i++
				} else {
					break
				}
			}
			tokens = append(tokens, token{tokenNumber, expr[start:i], start})
			continue
		}

		// Identifier or function name
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
			start := i
			i++
			for i < len(expr) {
				ch := expr[i]
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
					i++
				} else {
					break
				}
			}
			ident := strings.ToLower(expr[start:i])
			tokens = append(tokens, token{tokenIdentifier, ident, start})
			continue
		}

		// Operators
		if ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '%' || ch == '^' {
			tokens = append(tokens, token{tokenOperator, string(ch), i})
			i++
			continue
		}

		// Parentheses and comma
		if ch == '(' {
			tokens = append(tokens, token{tokenLeftParen, string(ch), i})
			i++
			continue
		}
		if ch == ')' {
			tokens = append(tokens, token{tokenRightParen, string(ch), i})
			i++
			continue
		}
		if ch == ',' {
			tokens = append(tokens, token{tokenComma, string(ch), i})
			i++
			continue
		}

		return nil, fmt.Errorf("unexpected character at position %d: %c", i, ch)
	}

	return tokens, nil
}

// expressionParser is a recursive descent parser for mathematical expressions.
type expressionParser struct {
	tokens []token
	pos    int
}

// parseExpression parses an expression (lowest precedence).
func (p *expressionParser) parseExpression() (float64, error) {
	return p.parseAddSub()
}

// parseAddSub parses addition and subtraction.
func (p *expressionParser) parseAddSub() (float64, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return 0, err
	}

	for p.pos < len(p.tokens) {
	 tok := p.tokens[p.pos]
		if tok.typ != tokenOperator {
			break
		}

		if tok.value == "+" || tok.value == "-" {
			p.pos++
			right, err := p.parseMulDiv()
			if err != nil {
				return 0, err
			}

			if tok.value == "+" {
				left += right
			} else {
				left -= right
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseMulDiv parses multiplication and division.
func (p *expressionParser) parseMulDiv() (float64, error) {
	left, err := p.parsePower()
	if err != nil {
		return 0, err
	}

	for p.pos < len(p.tokens) {
	 tok := p.tokens[p.pos]
		if tok.typ != tokenOperator {
			break
		}

		if tok.value == "*" || tok.value == "/" || tok.value == "%" {
			p.pos++
			right, err := p.parsePower()
			if err != nil {
				return 0, err
			}

			if tok.value == "*" {
				left *= right
			} else if tok.value == "/" {
				if right == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				left /= right
			} else {
				if right == 0 {
					return 0, fmt.Errorf("modulo by zero")
				}
				left = math.Mod(left, right)
			}
		} else {
			break
		}
	}

	return left, nil
}

// parsePower parses exponentiation.
func (p *expressionParser) parsePower() (float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}

	if p.pos < len(p.tokens) && p.tokens[p.pos].typ == tokenOperator && p.tokens[p.pos].value == "^" {
		p.pos++
		right, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		left = math.Pow(left, right)
	}

	return left, nil
}

// parseUnary parses unary operators.
func (p *expressionParser) parseUnary() (float64, error) {
	if p.pos < len(p.tokens) && p.tokens[p.pos].typ == tokenOperator {
		if p.tokens[p.pos].value == "-" {
			p.pos++
			val, err := p.parseUnary()
			if err != nil {
				return 0, err
			}
			return -val, nil
		}
		if p.tokens[p.pos].value == "+" {
			p.pos++
			return p.parseUnary()
		}
	}

	return p.parsePrimary()
}

// parsePrimary parses primary expressions (numbers, parentheses, functions).
func (p *expressionParser) parsePrimary() (float64, error) {
	if p.pos >= len(p.tokens) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	tok := p.tokens[p.pos]

	// Number
	if tok.typ == tokenNumber {
		p.pos++
		val, err := strconv.ParseFloat(tok.value, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", tok.value)
		}
		return val, nil
	}

	// Parenthesized expression
	if tok.typ == tokenLeftParen {
		p.pos++
		expr, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos].typ != tokenRightParen {
			return 0, fmt.Errorf("expected closing parenthesis")
		}
		p.pos++
		return expr, nil
	}

	// Function call or identifier
	if tok.typ == tokenIdentifier {
		// Check for constants
		if val, ok := constants[tok.value]; ok {
			p.pos++
			return val, nil
		}

		// Check for functions
		if fn, ok := functions[tok.value]; ok {
			p.pos++
			if p.pos >= len(p.tokens) || p.tokens[p.pos].typ != tokenLeftParen {
				return 0, fmt.Errorf("expected opening parenthesis after function %s", tok.value)
			}
			p.pos++

			// Parse arguments
			args := make([]float64, 0)
			if p.pos < len(p.tokens) && p.tokens[p.pos].typ != tokenRightParen {
				arg, err := p.parseExpression()
				if err != nil {
					return 0, err
				}
				args = append(args, arg)

				for p.pos < len(p.tokens) && p.tokens[p.pos].typ == tokenComma {
					p.pos++
					arg, err := p.parseExpression()
					if err != nil {
						return 0, err
					}
					args = append(args, arg)
				}
			}

			if p.pos >= len(p.tokens) || p.tokens[p.pos].typ != tokenRightParen {
				return 0, fmt.Errorf("expected closing parenthesis after function arguments")
			}
			p.pos++

			// Call function
			return fn(args)
		}

		return 0, fmt.Errorf("unknown identifier: %s", tok.value)
	}

	return 0, fmt.Errorf("unexpected token: %s", tok.value)
}

// constants maps constant names to their values.
var constants = map[string]float64{
	"pi":  math.Pi,
	"e":   math.E,
	"phi": 1.618033988749895,
}

// functionType represents a function signature.
type functionType func([]float64) (float64, error)

// functions maps function names to their implementations.
var functions = map[string]functionType{
	// Trigonometric functions
	"sin": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("sin expects 1 argument")
		}
		return math.Sin(args[0]), nil
	},
	"cos": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("cos expects 1 argument")
		}
		return math.Cos(args[0]), nil
	},
	"tan": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("tan expects 1 argument")
		}
		return math.Tan(args[0]), nil
	},
	"asin": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("asin expects 1 argument")
		}
		return math.Asin(args[0]), nil
	},
	"acos": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("acos expects 1 argument")
		}
		return math.Acos(args[0]), nil
	},
	"atan": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("atan expects 1 argument")
		}
		return math.Atan(args[0]), nil
	},
	"atan2": func(args []float64) (float64, error) {
		if len(args) != 2 {
			return 0, fmt.Errorf("atan2 expects 2 arguments")
		}
		return math.Atan2(args[0], args[1]), nil
	},

	// Hyperbolic functions
	"sinh": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("sinh expects 1 argument")
		}
		return math.Sinh(args[0]), nil
	},
	"cosh": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("cosh expects 1 argument")
		}
		return math.Cosh(args[0]), nil
	},
	"tanh": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("tanh expects 1 argument")
		}
		return math.Tanh(args[0]), nil
	},

	// Exponential and logarithmic functions
	"exp": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("exp expects 1 argument")
		}
		return math.Exp(args[0]), nil
	},
	"log": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("log expects 1 argument")
		}
		return math.Log(args[0]), nil
	},
	"log10": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("log10 expects 1 argument")
		}
		return math.Log10(args[0]), nil
	},
	"log2": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("log2 expects 1 argument")
		}
		return math.Log2(args[0]), nil
	},

	// Power and root functions
	"sqrt": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("sqrt expects 1 argument")
		}
		return math.Sqrt(args[0]), nil
	},
	"cbrt": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("cbrt expects 1 argument")
		}
		return math.Cbrt(args[0]), nil
	},

	// Rounding functions
	"ceil": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("ceil expects 1 argument")
		}
		return math.Ceil(args[0]), nil
	},
	"floor": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("floor expects 1 argument")
		}
		return math.Floor(args[0]), nil
	},
	"round": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("round expects 1 argument")
		}
		return math.Round(args[0]), nil
	},

	// Other functions
	"abs": func(args []float64) (float64, error) {
		if len(args) != 1 {
			return 0, fmt.Errorf("abs expects 1 argument")
		}
		return math.Abs(args[0]), nil
	},
	"min": func(args []float64) (float64, error) {
		if len(args) == 0 {
			return 0, fmt.Errorf("min expects at least 1 argument")
		}
		min := args[0]
		for _, v := range args[1:] {
			if v < min {
				min = v
			}
		}
		return min, nil
	},
	"max": func(args []float64) (float64, error) {
		if len(args) == 0 {
			return 0, fmt.Errorf("max expects at least 1 argument")
		}
		max := args[0]
		for _, v := range args[1:] {
			if v > max {
				max = v
			}
		}
		return max, nil
	},
	"pow": func(args []float64) (float64, error) {
		if len(args) != 2 {
			return 0, fmt.Errorf("pow expects 2 arguments")
		}
		return math.Pow(args[0], args[1]), nil
	},
}
