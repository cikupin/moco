package moco

import (
	"bufio"
	"crypto/md5"
	"fmt"

	"io"
	"strings"
)

const (
	BankMandiri = "mandiri"
	BankBCA     = "bca"
	BankBRI     = "bri"
)

type FailRecord struct {
	Raw  string
	Hash string
	Err  error
}

type reader struct {
	ioReader           io.Reader
	parser             Parser
	bankType           string
	mutations          []MutationBank
	unrecognizedRecord []FailRecord
}

type Reader interface {
	ReadMutation() error
	GetSuccess() []MutationBank
	GetFail() []FailRecord
}

func NewReader(ioReader io.Reader, bankType string) Reader {
	return &reader{
		ioReader: ioReader,
		parser:   NewParser(bankType),
		bankType: bankType,
	}
}

func (mr *reader) recordToMutation(rec []string) (*MutationBank, error) {
	if !mr.isValidRecord(rec) {
		return nil, fmt.Errorf("%v", "Not a valid record")
	}

	err := mr.parser.LoadRecord(rec)
	if err != nil {
		return nil, err
	}

	return mr.parser.GetMutation(), nil
}

func (mr *reader) isValidRecord(rec []string) bool {
	if mr.bankType == BankBCA {
		if len(rec) < 5 {
			return false
		}
	}
	if mr.bankType == BankBRI {
		if len(rec) < 13 {
			return false
		}
	}

	return true
}

func (mr *reader) getHash(s string) string {
	data := []byte(s)
	return fmt.Sprintf("%x", md5.Sum(data))
}

func (mr *reader) ReadMutation() error {
	scanner := bufio.NewScanner(mr.ioReader)
	for scanner.Scan() {
		textIn := scanner.Text()
		ioReader := strings.NewReader(textIn)
		csvReader := NewCsvReader(ioReader, ',')
		for {
			rec, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err == nil {
				mutation, err := mr.recordToMutation(rec)
				if mutation != nil {

					mr.appendToSuccessRecord(mutation)
				} else {
					mr.appendToErrorRecord(textIn, err)
				}
			} else {
				mr.appendToErrorRecord(textIn, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (mr *reader) appendToSuccessRecord(mutation *MutationBank) {
	mr.mutations = append(mr.mutations, *mutation)
}

func (mr *reader) appendToErrorRecord(s string, err error) {
	ur := new(FailRecord)
	ur.Raw = s
	ur.Hash = mr.getHash(s)
	ur.Err = err

	mr.unrecognizedRecord = append(mr.unrecognizedRecord, *ur)
}

func (mr *reader) GetSuccess() []MutationBank {
	return mr.mutations
}

func (mr *reader) GetFail() []FailRecord {
	return mr.unrecognizedRecord
}