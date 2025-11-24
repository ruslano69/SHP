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
	"log"
	"net/http"
	"os"
	"time"
)

// SHP Simple - No Parsing Architecture
// Version: 2.0.0
// Description: Signed HTML by raw bytes - no canonicalization required!

var (
	genKeys = flag.Bool("genkeys", false, "Generate RSA keypair")
	serve   = flag.Bool("serve", false, "Start SHP server")
	port    = flag.String("port", "8080", "Server port")
	privKeyFile = flag.String("key", "private.pem", "Private key file")
	pubKeyFile = flag.String("pub", "public.pem", "Public key file")
	
	// Global private key (loaded once at startup)
	serverPrivateKey *rsa.PrivateKey
)

// Simple XHTML Generator (no parsing!)
func generateValidXHTML(title, content string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" 
  "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>%s</title>
  <meta http-equiv="Content-Type" content="application/xhtml+xml; charset=utf-8" />
</head>
<body>
%s
</body>
</html>`, title, content)
}

// Generate demo content
func generateDemoContent() string {
	return `
  <h1>SHP Protocol Demo - Raw Bytes Signing</h1>
  <p>This page demonstrates the revolutionary SHP concept:</p>
  <ul>
    <li><strong>No Parsing Required:</strong> Server signs raw XHTML bytes</li>
    <li><strong>No Canonicalization:</strong> Simple hash of exact bytes</li>
    <li><strong>Guaranteed Integrity:</strong> Any change = invalid signature</li>
    <li><strong>Automatic Strict Mode:</strong> Content-Type: application/xhtml+xml</li>
  </ul>
  
  <h2>How It Works</h2>
  <p>The server generates valid XHTML and signs the exact bytes. The Service Worker 
  verifies these bytes without any DOM parsing. If the signature is valid, 
  the browser receives strict XHTML parsing.</p>
  
  <div style="background: #e8f5e8; padding: 15px; border: 2px solid #4CAF50; border-radius: 5px;">
    <h3>🛡️ Security Guarantees</h3>
    <ul>
      <li>✅ Content unmodified (cryptographic proof)</li>
      <li>✅ Structure was valid XHTML (server guarantees)</li>
      <li>✅ Automatic strict parsing mode</li>
      <li>✅ CDN injection protection</li>
    </ul>
  </div>
  
  <h2>Attack Scenarios</h2>
  
  <h3>CDN Script Injection:</h3>
  <pre style="background: #f5f5f5; padding: 10px; border-left: 4px solid #f44336;">
Original: &lt;body&gt;&lt;p&gt;Content&lt;/p&gt;&lt;/body&gt;
Injected: &lt;body&gt;&lt;p&gt;Content&lt;/p&gt;&lt;script&gt;evil()&lt;/script&gt;&lt;/body&gt;
Result:   SHA-256 different → signature INVALID → BLOCKED
  </pre>
  
  <h3>Broken HTML from Server:</h3>
  <pre style="background: #f5f5f5; padding: 10px; border-left: 4px solid #FF9800;">
Server:   Signs broken HTML as-is
Browser:  Receives application/xhtml+xml
Parsing:  Fails on broken XML → Yellow Screen of Death
Result:   Admin gets notified → Fixes server code
  </pre>
  
  <p style="margin-top: 30px; font-style: italic; color: #666;">
    This approach eliminates the complexity of HTML canonicalization while 
    providing stronger security guarantees.
  </p>
`
}

// Sign raw bytes (no parsing, no canonicalization!)
func signRawBytes(data []byte, privateKey *rsa.PrivateKey) (string, error) {
	hash := sha256.Sum256(data)
	signatureBytes, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signatureBytes), nil
}

// HTTP Handler for SHP
func shpHandler(w http.ResponseWriter, r *http.Request) {
	// Generate valid XHTML content
	title := "SHP Raw Bytes Demo"
	content := generateDemoContent()
	xhtml := generateValidXHTML(title, content)
	xhtmlBytes := []byte(xhtml)

	// Sign RAW BYTES (the genius part!)
	signature, err := signRawBytes(xhtmlBytes, serverPrivateKey)
	if err != nil {
		http.Error(w, "Signing error", 500)
		log.Printf("Signing error: %v", err)
		return
	}

	// Set SHP headers
	w.Header().Set("SHP-Signature", signature)
	w.Header().Set("SHP-Algorithm", "SHA256-RSA2048")
	w.Header().Set("SHP-Version", "2.0")
	w.Header().Set("SHP-Timestamp", time.Now().UTC().Format(time.RFC3339))
	
	// Critical: Force strict XHTML parsing
	w.Header().Set("Content-Type", "application/xhtml+xml")

	// Return signed XHTML
	w.Write(xhtmlBytes)

	log.Printf("Served signed XHTML: %d bytes, signature: %s...", 
		len(xhtmlBytes), signature[:32])
}

// Generate a simple XHTML page for testing
func generateSimpleXHTML() {
	xhtml := generateValidXHTML("Simple Test Page", 
		`<h1>Simple XHTML Test</h1><p>This is a simple test page.</p>`)
	
	err := os.WriteFile("simple.html", []byte(xhtml), 0644)
	if err != nil {
		log.Fatalf("Failed to write simple.html: %v", err)
	}
	fmt.Println("Generated: simple.html")
}

// Generate RSA keypair
func generateKeypair(privFile, pubFile string) {
	fmt.Println("🔑 Generating RSA-2048 keypair...")

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}

	// Save Private Key
	privBytes := x509.MarshalPKCS1PrivateKey(key)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})
	if err := os.WriteFile(privFile, privPEM, 0600); err != nil {
		log.Fatalf("Failed to save private key: %v", err)
	}

	// Save Public Key
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		log.Fatalf("Failed to marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	if err := os.WriteFile(pubFile, pubPEM, 0644); err != nil {
		log.Fatalf("Failed to save public key: %v", err)
	}

	fmt.Printf("✅ Keys generated:\n   Private: %s\n   Public:  %s\n", privFile, pubFile)
}

// Load private key
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// Start SHP server
func startServer() {
	var err error
	serverPrivateKey, err = loadPrivateKey(*privKeyFile)
	if err != nil {
		log.Fatalf("Failed to load private key %s: %v", *privKeyFile, err)
	}

	fmt.Printf("🚀 Starting SHP Simple Server v2.0\n")
	fmt.Printf("   Port: %s\n", *port)
	fmt.Printf("   Key:  %s\n", *privKeyFile)
	fmt.Printf("   Mode: Raw Bytes Signing (No Parsing!)\n")
	fmt.Println("   Server ready! Open browser to see SHP in action.")
	
	// Register handler
	http.HandleFunc("/", shpHandler)
	
	// Start server
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func main() {
	flag.Parse()

	if *genKeys {
		generateKeypair(*privKeyFile, *pubKeyFile)
		return
	}

	if *serve {
		startServer()
		return
	}

	// Generate simple test file
	generateSimpleXHTML()

	fmt.Println("🔧 SHP Simple v2.0 - No Parsing Architecture")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  -genkeys    Generate RSA keypair")
	fmt.Println("  -serve      Start SHP server")
	fmt.Println("  -port 8080  Server port (default: 8080)")
	fmt.Println("  -key file   Private key file")
	fmt.Println("  -pub file   Public key file")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ./shp_simple -genkeys")
	fmt.Println("  ./shp_simple -serve -port 8080")
}
