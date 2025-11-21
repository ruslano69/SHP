package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	genKeys := flag.Bool("genkeys", false, "Generate new RSA keypair")
	signFile := flag.String("sign", "", "Sign HTML file")
	privKeyFile := flag.String("key", "private.pem", "Private key file")
	pubKeyFile := flag.String("pub", "public.pem", "Public key file")
	
	flag.Parse()

	if *genKeys {
		generateKeypair(*privKeyFile, *pubKeyFile)
		return
	}

	if *signFile != "" {
		signHTMLFile(*signFile, *privKeyFile, *pubKeyFile)
		return
	}

	flag.Usage()
}

// Generate RSA keypair
func generateKeypair(privFile, pubFile string) {
	fmt.Println("🔐 Generating RSA-2048 keypair...")

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating key: %v\n", err)
		os.Exit(1)
	}

	// Save private key
	privPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privOut, err := os.Create(privFile)
	if err != nil {
		fmt.Printf("Error creating private key file: %v\n", err)
		os.Exit(1)
	}
	pem.Encode(privOut, privPEM)
	privOut.Close()

	// Save public key
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		fmt.Printf("Error marshaling public key: %v\n", err)
		os.Exit(1)
	}
	pubPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pubOut, err := os.Create(pubFile)
	if err != nil {
		fmt.Printf("Error creating public key file: %v\n", err)
		os.Exit(1)
	}
	pem.Encode(pubOut, pubPEM)
	pubOut.Close()

	fmt.Printf("✅ Keys generated:\n")
	fmt.Printf("   Private: %s\n", privFile)
	fmt.Printf("   Public: %s\n", pubFile)
}

// Sign HTML file and inject metadata
func signHTMLFile(htmlFile, privFile, pubFile string) {
	fmt.Printf("📝 Signing %s...\n", htmlFile)

	// Read HTML
	htmlContent, err := ioutil.ReadFile(htmlFile)
	if err != nil {
		fmt.Printf("Error reading HTML: %v\n", err)
		os.Exit(1)
	}

	// Load private key
	privPEM, err := ioutil.ReadFile(privFile)
	if err != nil {
		fmt.Printf("Error reading private key: %v\n", err)
		os.Exit(1)
	}
	block, _ := pem.Decode(privPEM)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Printf("Error parsing private key: %v\n", err)
		os.Exit(1)
	}

	// Load public key
	pubPEM, err := ioutil.ReadFile(pubFile)
	if err != nil {
		fmt.Printf("Error reading public key: %v\n", err)
		os.Exit(1)
	}
	pubBlock, _ := pem.Decode(pubPEM)
	pubKeyBase64 := base64.StdEncoding.EncodeToString(pubBlock.Bytes)

	// Create canonical content (simplified - remove script tags for signing)
	canonical := string(htmlContent)
	canonical = removeScriptTags(canonical)

	// Hash content
	hash := sha256.Sum256([]byte(canonical))

	// Sign
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		fmt.Printf("Error signing: %v\n", err)
		os.Exit(1)
	}
	signatureBase64 := base64.StdEncoding.EncodeToString(signature)

	// Inject metadata into HTML
	htmlStr := string(htmlContent)
	htmlStr = strings.Replace(htmlStr, 
		`<meta name="shp-signature" content="PLACEHOLDER_BASE64_SIGNATURE">`,
		fmt.Sprintf(`<meta name="shp-signature" content="%s">`, signatureBase64), 1)
	htmlStr = strings.Replace(htmlStr,
		`<meta name="shp-pubkey" content="PLACEHOLDER_BASE64_PUBKEY">`,
		fmt.Sprintf(`<meta name="shp-pubkey" content="%s">`, pubKeyBase64), 1)

	// Write signed HTML
	signedFile := strings.Replace(htmlFile, ".html", "-signed.html", 1)
	err = ioutil.WriteFile(signedFile, []byte(htmlStr), 0644)
	if err != nil {
		fmt.Printf("Error writing signed HTML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Signed HTML created: %s\n", signedFile)
	fmt.Printf("   Signature: %s...\n", signatureBase64[:40])
	fmt.Printf("   Public Key: %s...\n", pubKeyBase64[:40])
}

// Remove script tags for canonical representation
func removeScriptTags(html string) string {
	// Simple regex replacement (in production use proper HTML parser)
	result := html
	for strings.Contains(result, "<script") {
		start := strings.Index(result, "<script")
		end := strings.Index(result[start:], "</script>")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+9:]
	}
	return result
}
