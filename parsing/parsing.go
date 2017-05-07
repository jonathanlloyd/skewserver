package parsing

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Custom error types for package

type ParseError struct{ message string }

func (e ParseError) Error() string {
	return fmt.Sprintf("Failed trying to parse STOMP frame: %s", e.message)
}

// STOMP Frame Parser
// Parses STOMP message frames from a bufio.Reader

type StompParser struct {
	stream         ReadPeeker
	reachedEOF     bool
	frameJustEnded bool
}

func NewStompParserFromReader(reader io.Reader) (parser StompParser) {
	bufferedReader := bufio.NewReader(reader)
	return StompParser{stream: bufferedReader}
}

// Parsing

type Frame struct {
	Command CommandType
	Headers map[string]string
	Body    []byte
}

type CommandType int

const (
	SEND        CommandType = iota + 1
	SUBSCRIBE   CommandType = iota + 1
	UNSUBSCRIBE CommandType = iota + 1
	BEGIN       CommandType = iota + 1
	COMMIT      CommandType = iota + 1
	ABORT       CommandType = iota + 1
	ACK         CommandType = iota + 1
	NACK        CommandType = iota + 1
	DISCONNECT  CommandType = iota + 1
	CONNECT     CommandType = iota + 1
	STOMP       CommandType = iota + 1
	CONNECTED   CommandType = iota + 1
	MESSAGE     CommandType = iota + 1
	RECEIPT     CommandType = iota + 1
	ERROR       CommandType = iota + 1
)

var commands = map[string]CommandType{
	"SEND":        SEND,
	"SUBSCRIBE":   SUBSCRIBE,
	"UNSUBSCRIBE": UNSUBSCRIBE,
	"BEGIN":       BEGIN,
	"COMMIT":      COMMIT,
	"ABORT":       ABORT,
	"ACK":         ACK,
	"NACK":        NACK,
	"DISCONNECT":  DISCONNECT,
	"CONNECT":     CONNECT,
	"STOMP":       STOMP,
	"CONNECTED":   CONNECTED,
	"MESSAGE":     MESSAGE,
	"RECEIPT":     RECEIPT,
	"ERROR":       ERROR,
}

func (parser *StompParser) NextFrame() (parsedFrame Frame, err error) {
	//Command
	tokType, tokLiteral := parser.nextToken()
	if tokType != COMMAND && !parser.reachedEOF {
		return Frame{}, ParseError{message: "Frame must begin with a command"}
	}
	command := commands[string(tokLiteral)]

	//Headers
	tokType, tokLiteral = parser.nextToken() // Could be header or body

	headers := map[string]string{}
	for ; tokType == HEADER_KEY; tokType, tokLiteral = parser.nextToken() {
		if tokType == HEADER_KEY {
			header_key := string(tokLiteral)
			tokType, tokLiteral = parser.nextToken()
			if tokType != HEADER_VALUE && !parser.reachedEOF {
				return Frame{}, ParseError{message: "Headers must have values"}
			}
			header_value := string(tokLiteral)
			headers[header_key] = header_value
		} else {
			break
		}
	}

	//Body
	if tokType != BODY && !parser.reachedEOF {
		return Frame{}, ParseError{message: "Frames must contain bodies"}
	}
	body := tokLiteral

	// If we have reached the end of the stream before we have parsed a valid
	// frame then no more tokens can be returned.
	if parser.reachedEOF {
		return Frame{}, io.EOF
	}

	//Delimiter
	tokType, tokLiteral = parser.nextToken()
	if tokType != DELIMITER && !parser.reachedEOF {
		return Frame{}, ParseError{message: "Frames must end with a null byte"}
	}

	return Frame{Command: command, Headers: headers, Body: body}, nil
}

// Scanning / lexing

type TokenType int

const (
	NULL_TOKEN TokenType = iota + 1
	COMMAND
	HEADER_KEY
	HEADER_VALUE
	BODY
	DELIMITER
	INVALID_TOKEN
)

type TerminatorType int

const (
	EOL TerminatorType = iota + 1
	HEADER_SEPARATOR
)

type ReadPeeker interface {
	UnreadByte() error
	ReadByte() (byte, error)
	Peek(int) ([]byte, error)
}

// Parse the byte stream against the following rules (in order)
//  - <EOL>(.*)[NULL] = BODY
//  - (NULL)EOL* = DELIMETER
//  - :(HEADER_STR)EOL = HEADER_VALUE
//  - (COMMAND_STR)EOL = COMMAND
//  - (HEADER_STR): = HEADER_KEY
//  - (.*) = INVALID_TOKEN
func (parser *StompParser) nextToken() (tokType TokenType, tokLiteral []byte) {
	var terminator TerminatorType

	if parser.frameJustEnded {
		parser.skipEOLs()
		parser.frameJustEnded = false
	}

	peekBytes, err := parser.stream.Peek(1)
	if err != nil {
		parser.reachedEOF = true
		return NULL_TOKEN, []byte{}
	}
	currentByte := peekBytes[0]

	switch {
	case currentByte == '\x00':
		tokType = DELIMITER
		tokLiteral = []byte{currentByte}
		parser.stream.ReadByte()
		parser.frameJustEnded = true
	case currentByte == '\r' || currentByte == '\n':
		foundEOL := parser.scanEOL()
		if foundEOL {
			tokType = BODY
			tokLiteral = parser.scanTillDelimiter()
		} else {
			tokType = INVALID_TOKEN
		}
	case currentByte == ':':
		parser.stream.ReadByte()
		tokLiteral, terminator = parser.scanTillTerminator()
		if terminator == EOL {
			tokType = HEADER_VALUE
		} else {
			tokType = INVALID_TOKEN
		}
	default:
		tokLiteral, terminator = parser.scanTillTerminator()
		switch {
		case isCommand(tokLiteral) && terminator == EOL:
			tokType = COMMAND
		case terminator == HEADER_SEPARATOR:
			tokType = HEADER_KEY
		default:
			tokType = INVALID_TOKEN
		}
	}

	return tokType, tokLiteral
}

func (parser *StompParser) skipEOLs() {
	for {
		if !parser.scanEOL() {
			break
		}
	}
}

func (parser *StompParser) scanEOL() (found bool) {
	peekBytes, err := parser.stream.Peek(2)
	if err != nil {
		parser.reachedEOF = true
		return false
	}

	if peekBytes[0] == '\n' {
		found = true
		parser.stream.ReadByte()
	} else if bytes.Equal(peekBytes, []byte{'\r', '\n'}) {
		found = true
		parser.stream.ReadByte()
		parser.stream.ReadByte()
	} else {
		found = false
	}
	return
}

func (parser *StompParser) scanHeaderSeparator() (found bool) {
	peekBytes, err := parser.stream.Peek(1)
	if err != nil {
		parser.reachedEOF = true
		return false
	}

	if peekBytes[0] == ':' {
		found = true
	} else {
		found = false
	}

	return
}

func (parser *StompParser) scanTillDelimiter() (literal []byte) {
	for {
		peekBytes, err := parser.stream.Peek(1)
		if err != nil {
			parser.reachedEOF = true
			break
		} else if peekBytes[0] == '\x00' {
			break
		} else {
			currentByte, err := parser.stream.ReadByte()
			if err != nil {
				parser.reachedEOF = true
				break
			}
			literal = append(literal, currentByte)
		}
	}
	return
}

func (parser *StompParser) scanTillTerminator() (literal []byte, term TerminatorType) {
	literal = []byte{}

	for term == 0 && !parser.reachedEOF {
		switch {
		case parser.scanEOL():
			term = EOL
		case parser.scanHeaderSeparator():
			term = HEADER_SEPARATOR
		default:
			currentByte, err := parser.stream.ReadByte()
			if err != nil {
				parser.reachedEOF = true
				break
			}
			literal = append(literal, currentByte)
		}
	}

	return
}

func isCommand(literal []byte) (result bool) {
	_, result = commands[string(literal)]
	return
}
