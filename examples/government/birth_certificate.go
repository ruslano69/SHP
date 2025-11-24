// Government Birth Certificate Example
// This demonstrates how to generate a signed birth certificate using SHP v2.0

package main

import (
	"fmt"
	"time"
)

// BirthCertificateData represents the data needed for a birth certificate
type BirthCertificateData struct {
	CertificateID string
	FullName      string
	DateOfBirth   string
	PlaceOfBirth  string
	MotherName    string
	FatherName    string
	RegistrationNo string
	IssueDate     string
}

// GenerateBirthCertificate creates XHTML for a birth certificate
func GenerateBirthCertificate(data BirthCertificateData) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" 
  "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>Birth Certificate - %s</title>
  <meta http-equiv="Content-Type" content="application/xhtml+xml; charset=utf-8" />
  <meta name="document-type" content="birth-certificate" />
  <meta name="document-id" content="%s" />
  <meta name="issue-date" content="%s" />
  <style type="text/css">
    body {
      font-family: 'Times New Roman', serif;
      max-width: 800px;
      margin: 40px auto;
      padding: 40px;
      background: #f5f5f5;
    }
    .certificate {
      background: white;
      padding: 60px;
      border: 10px double #003366;
      box-shadow: 0 4px 6px rgba(0,0,0,0.1);
    }
    .header {
      text-align: center;
      margin-bottom: 40px;
    }
    .seal {
      font-size: 3em;
      color: #003366;
    }
    h1 {
      color: #003366;
      font-size: 2.5em;
      margin: 20px 0;
    }
    .data-table {
      width: 100%%;
      margin: 30px 0;
      border-collapse: collapse;
    }
    .data-table td {
      padding: 15px;
      border-bottom: 1px solid #ddd;
    }
    .data-table td:first-child {
      font-weight: bold;
      width: 40%%;
      color: #003366;
    }
    .signature-block {
      margin-top: 60px;
      padding-top: 20px;
      border-top: 2px solid #003366;
    }
    .verification {
      background: #e8f5e9;
      padding: 20px;
      margin-top: 40px;
      border-left: 4px solid #4caf50;
    }
  </style>
</head>
<body>
  <div class="certificate">
    <div class="header">
      <div class="seal">🇺🇦</div>
      <h1>BIRTH CERTIFICATE</h1>
      <p style="font-size: 1.2em;">Ukraine - Ministry of Interior</p>
      <p>Civil Registration Department</p>
    </div>
    
    <table class="data-table">
      <tr>
        <td>Certificate Number:</td>
        <td>%s</td>
      </tr>
      <tr>
        <td>Full Name:</td>
        <td><strong style="font-size: 1.2em;">%s</strong></td>
      </tr>
      <tr>
        <td>Date of Birth:</td>
        <td>%s</td>
      </tr>
      <tr>
        <td>Place of Birth:</td>
        <td>%s</td>
      </tr>
      <tr>
        <td>Mother's Name:</td>
        <td>%s</td>
      </tr>
      <tr>
        <td>Father's Name:</td>
        <td>%s</td>
      </tr>
      <tr>
        <td>Registration Number:</td>
        <td>%s</td>
      </tr>
    </table>
    
    <div class="signature-block">
      <p><strong>Issued by:</strong> Civil Registration Office, Kyiv</p>
      <p><strong>Issue Date:</strong> %s</p>
      <p><strong>Registrar:</strong> Olena Kovalenko</p>
    </div>
    
    <div class="verification">
      <p><strong>✅ This certificate is digitally signed</strong></p>
      <p>Signature algorithm: SHA-256 with RSA-2048</p>
      <p>Issuing authority: Ministry of Interior, Ukraine</p>
      <p>Certificate ID: %s</p>
      <p><em>This document can be verified instantly by any browser with SHP support.</em></p>
      <p><em>No additional software or apps required.</em></p>
    </div>
  </div>
</body>
</html>`, 
		data.CertificateID, // title
		data.CertificateID, // meta
		data.IssueDate,     // meta
		data.CertificateID, // table
		data.FullName,
		data.DateOfBirth,
		data.PlaceOfBirth,
		data.MotherName,
		data.FatherName,
		data.RegistrationNo,
		data.IssueDate,
		data.CertificateID,
	)
}

// Example usage
func main() {
	// Sample data
	certificate := BirthCertificateData{
		CertificateID:  "BC-2025-123456",
		FullName:       "ANNA OLEKSANDRIVNA SHEVCHENKO",
		DateOfBirth:    "January 15, 2025",
		PlaceOfBirth:   "Kyiv, Ukraine",
		MotherName:     "Olena Ivanivna Shevchenko",
		FatherName:     "Oleksandr Petrovych Shevchenko",
		RegistrationNo: "REG-UA-2025-001234",
		IssueDate:      time.Now().Format("January 02, 2006"),
	}
	
	// Generate XHTML
	xhtml := GenerateBirthCertificate(certificate)
	
	// In real application:
	// 1. Sign this XHTML with private key
	// 2. Send with SHP headers
	// 3. Service Worker verifies signature
	// 4. Browser displays verified certificate
	
	fmt.Println(xhtml)
}

/*
Integration with main SHP server:

func birthCertificateHandler(w http.ResponseWriter, r *http.Request) {
    // Get certificate ID from URL
    certID := r.URL.Query().Get("id")
    
    // Fetch certificate data from database
    data := fetchCertificateData(certID)
    
    // Generate XHTML
    xhtml := GenerateBirthCertificate(data)
    
    // Sign and send (reuse from shp_simple.go)
    signature, _ := signRawBytes([]byte(xhtml), serverPrivateKey)
    
    w.Header().Set("SHP-Signature", signature)
    w.Header().Set("SHP-Algorithm", "SHA256-RSA2048")
    w.Header().Set("SHP-Version", "2.0")
    w.Header().Set("SHP-Document-ID", certID)
    w.Header().Set("Content-Type", "application/xhtml+xml")
    
    w.Write([]byte(xhtml))
}
*/
