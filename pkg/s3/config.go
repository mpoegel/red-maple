package s3

import "time"

// ClientOption is a functional option for configuring the S3 client.
type ClientOption func(*Client)

// WithEndpoint sets the S3-compatible endpoint URL.
// If not set, defaults to s3.{Region}.amazonaws.com for AWS S3.
func WithEndpoint(endpoint string) ClientOption {
	return func(c *Client) {
		c.endpoint = endpoint
	}
}

// WithRegion sets the AWS region for the S3 bucket.
// Defaults to "us-east-1".
func WithRegion(region string) ClientOption {
	return func(c *Client) {
		c.region = region
	}
}

// WithBucket sets the S3 bucket name.
func WithBucket(bucket string) ClientOption {
	return func(c *Client) {
		c.bucket = bucket
	}
}

// WithCredentials sets the AWS access key and secret key for authentication.
func WithCredentials(accessKey, secretKey string) ClientOption {
	return func(c *Client) {
		c.accessKey = accessKey
		c.secretKey = secretKey
	}
}

// WithRetentionDays sets the number of days to retain data before cleanup.
// Defaults to 30.
func WithRetentionDays(days int) ClientOption {
	return func(c *Client) {
		c.retentionDays = days
	}
}

// WithFlushInterval sets the interval for flushing buffered data to S3.
// Defaults to 1 minute.
func WithFlushInterval(interval time.Duration) ClientOption {
	return func(c *Client) {
		c.buffer.interval = interval
	}
}

// WithScheme sets the URL scheme for the S3 endpoint.
// Defaults to "https". Use "http" for local MinIO/Garage instances.
func WithScheme(scheme string) ClientOption {
	return func(c *Client) {
		c.scheme = scheme
	}
}
