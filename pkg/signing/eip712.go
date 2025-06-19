package signing

import (
	"fmt"
	"math/big"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pooofdevelopment/go-clob-client/pkg/signer"
)

// Constants for CLOB authentication
// Based on: py-clob-client-main/py_clob_client/signing/eip712.py:8-10
const (
	CLOB_DOMAIN_NAME = "ClobAuthDomain"
	CLOB_VERSION     = "1"
	MSG_TO_SIGN      = "This message attests that I control the given wallet"
)

// ClobAuth represents the authentication message structure
// Based on: py-clob-client-main/py_clob_client/signing/model.py:1-14
type ClobAuth struct {
	Address   string `json:"address"`
	Timestamp string `json:"timestamp"`
	Nonce     int    `json:"nonce"`
	Message   string `json:"message"`
}

// EIP712Domain represents the domain separator for EIP712
// Based on: go-order-utils-main/pkg/eip712/constants.go and py-clob-client-main/py_clob_client/signing/eip712.py:13-14
type EIP712Domain struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	ChainID *big.Int       `json:"chainId"`
}

// SignClobAuthMessage signs the CLOB authentication message
// Based on: py-clob-client-main/py_clob_client/signing/eip712.py:17-28
func SignClobAuthMessage(s *signer.Signer, timestamp int64, nonce int) (string, error) {
	// Create the auth message
	authMsg := ClobAuth{
		Address:   s.Address(),
		Timestamp: fmt.Sprintf("%d", timestamp),
		Nonce:     nonce,
		Message:   MSG_TO_SIGN,
	}
	
	// Create domain separator
	// Based on: go-order-utils-main/pkg/eip712/eip712.go:12-27
	domain := EIP712Domain{
		Name:    CLOB_DOMAIN_NAME,
		Version: CLOB_VERSION,
		ChainID: big.NewInt(int64(s.GetChainID())),
	}
	
	// Build the domain separator hash
	domainSeparator := buildDomainSeparatorHash(domain)
	
	// Build the message hash
	messageHash := buildMessageHash(authMsg)
	
	// Combine according to EIP712 standard
	// Based on: go-order-utils-main/pkg/eip712/eip712.go:45-52
	rawData := append([]byte("\x19\x01"), domainSeparator[:]...)
	rawData = append(rawData, messageHash[:]...)
	finalHash := crypto.Keccak256Hash(rawData)
	
	// Sign the hash
	signature, err := s.Sign(finalHash.Bytes())
	if err != nil {
		return "", err
	}
	
	return signature, nil
}

// buildDomainSeparatorHash builds the domain separator hash
// Based on: go-order-utils-main/pkg/eip712/eip712.go:12-27
func buildDomainSeparatorHash(domain EIP712Domain) common.Hash {
	// EIP712Domain type hash
	// keccak256("EIP712Domain(string name,string version,uint256 chainId)")
	typeHash := crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId)"))
	
	// Hash the domain values
	nameHash := crypto.Keccak256Hash([]byte(domain.Name))
	versionHash := crypto.Keccak256Hash([]byte(domain.Version))
	
	// Encode and hash
	// Based on ABI encoding from go-order-utils-main/pkg/eip712/encode.go
	encoded := make([]byte, 0, 128)
	encoded = append(encoded, typeHash.Bytes()...)
	encoded = append(encoded, nameHash.Bytes()...)
	encoded = append(encoded, versionHash.Bytes()...)
	encoded = append(encoded, common.LeftPadBytes(domain.ChainID.Bytes(), 32)...)
	
	return crypto.Keccak256Hash(encoded)
}

// buildMessageHash builds the message hash for ClobAuth
// Based on: py-clob-client-main/py_clob_client/signing/model.py and EIP712 standard
func buildMessageHash(auth ClobAuth) common.Hash {
	// ClobAuth type hash
	// keccak256("ClobAuth(address address,string timestamp,uint256 nonce,string message)")
	typeHash := crypto.Keccak256Hash([]byte("ClobAuth(address address,string timestamp,uint256 nonce,string message)"))
	
	// Hash string values
	timestampHash := crypto.Keccak256Hash([]byte(auth.Timestamp))
	messageHash := crypto.Keccak256Hash([]byte(auth.Message))
	
	// Convert nonce to big.Int
	nonceBig := big.NewInt(int64(auth.Nonce))
	
	// Encode and hash
	encoded := make([]byte, 0, 160)
	encoded = append(encoded, typeHash.Bytes()...)
	// Address should be 20 bytes, left-padded to 32
	addressBytes := common.HexToAddress(auth.Address).Bytes()
	encoded = append(encoded, common.LeftPadBytes(addressBytes, 32)...)
	encoded = append(encoded, timestampHash.Bytes()...)
	encoded = append(encoded, common.LeftPadBytes(nonceBig.Bytes(), 32)...)
	encoded = append(encoded, messageHash.Bytes()...)
	
	return crypto.Keccak256Hash(encoded)
}