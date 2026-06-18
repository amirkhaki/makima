package dsl

type TokenType int

const (
	TokenWhen TokenType = iota
	TokenEntering
	TokenThen
	TokenBrowser
	TokenCDP
	TokenHyprctl
	TokenPopup
	TokenNotify
	TokenExec
	TokenCategory
	TokenMatch
	TokenMatches
	TokenGrace
	TokenCooldown
	TokenBudget
	TokenMaxPerHour
	TokenMaxPerDay
	TokenMaxPerWeek
	TokenIdent
	TokenString
	TokenNumber
	TokenDuration
	TokenDot
	TokenLBrace
	TokenRBrace
	TokenLBracket
	TokenRBracket
	TokenEOF
)

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
	}
}

func (l *Lexer) Tokenize() []Token {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]

		switch {
		case ch == '#':
			l.skipComment()
		case ch == '"':
			l.readString()
		case ch == '.':
			l.tokens = append(l.tokens, Token{Type: TokenDot, Value: "."})
			l.pos++
		case ch == '{':
			l.tokens = append(l.tokens, Token{Type: TokenLBrace, Value: "{"})
			l.pos++
		case ch == '}':
			l.tokens = append(l.tokens, Token{Type: TokenRBrace, Value: "}"})
			l.pos++
		case ch == '[':
			l.tokens = append(l.tokens, Token{Type: TokenLBracket, Value: "["})
			l.pos++
		case ch == ']':
			l.tokens = append(l.tokens, Token{Type: TokenRBracket, Value: "]"})
			l.pos++
		case isDigit(ch):
			l.readNumberOrDuration()
		case isAlpha(ch):
			l.readIdent()
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			l.pos++
		default:
			l.pos++
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokenEOF, Value: ""})
	return l.tokens
}

func (l *Lexer) skipComment() {
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.pos++
	}
}

func (l *Lexer) readString() {
	l.pos++
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		l.pos++
	}
	value := l.input[start:l.pos]
	if l.pos < len(l.input) {
		l.pos++
	}
	l.tokens = append(l.tokens, Token{Type: TokenString, Value: value})
}

func (l *Lexer) readNumberOrDuration() {
	start := l.pos
	for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
		l.pos++
	}
	if l.pos < len(l.input) && (l.input[l.pos] == 's' || l.input[l.pos] == 'm' || l.input[l.pos] == 'h') {
		l.pos++
		l.tokens = append(l.tokens, Token{Type: TokenDuration, Value: l.input[start:l.pos]})
	} else {
		l.tokens = append(l.tokens, Token{Type: TokenNumber, Value: l.input[start:l.pos]})
	}
}

func (l *Lexer) readIdent() {
	start := l.pos
	for l.pos < len(l.input) && (isAlphaNumeric(l.input[l.pos]) || l.input[l.pos] == '-') {
		l.pos++
	}
	word := l.input[start:l.pos]

	var tokType TokenType
	switch word {
	case "when":
		tokType = TokenWhen
	case "entering":
		tokType = TokenEntering
	case "then":
		tokType = TokenThen
	case "browser":
		tokType = TokenBrowser
	case "cdp":
		tokType = TokenCDP
	case "hyprctl":
		tokType = TokenHyprctl
	case "popup":
		tokType = TokenPopup
	case "notify":
		tokType = TokenNotify
	case "exec":
		tokType = TokenExec
	case "category":
		tokType = TokenCategory
	case "match":
		tokType = TokenMatch
	case "matches":
		tokType = TokenMatches
	case "grace":
		tokType = TokenGrace
	case "cooldown":
		tokType = TokenCooldown
	case "budget":
		tokType = TokenBudget
	case "max_per_hour":
		tokType = TokenMaxPerHour
	case "max_per_day":
		tokType = TokenMaxPerDay
	case "max_per_week":
		tokType = TokenMaxPerWeek
	default:
		tokType = TokenIdent
	}

	l.tokens = append(l.tokens, Token{Type: tokType, Value: word})
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isAlphaNumeric(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}


