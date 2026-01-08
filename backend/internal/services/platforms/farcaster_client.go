package platforms

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FarcasterClient implements PlatformAdapter for Farcaster (via Neynar/Hubble APIs)
type FarcasterClient struct {
	creds         *AccountCredentials
	httpClient    *http.Client
	neynarAPIKey  string
	neynarBaseURL string
	hubbleURL     string
	authenticated bool
	signerKey     ed25519.PrivateKey
}

// Neynar API response structures
type NeynarUser struct {
	FID            uint64 `json:"fid"`
	Username       string `json:"username"`
	DisplayName    string `json:"display_name"`
	PfpURL         string `json:"pfp_url"`
	Bio            string `json:"profile,omitempty"`
	FollowerCount  int    `json:"follower_count"`
	FollowingCount int    `json:"following_count"`
	Verifications  []string `json:"verifications"`
}

type NeynarCast struct {
	Hash       string     `json:"hash"`
	ParentHash string     `json:"parent_hash,omitempty"`
	Author     NeynarUser `json:"author"`
	Text       string     `json:"text"`
	Timestamp  string     `json:"timestamp"`
	Reactions  struct {
		Likes   int `json:"likes_count"`
		Recasts int `json:"recasts_count"`
	} `json:"reactions"`
	Replies struct {
		Count int `json:"count"`
	} `json:"replies"`
	Embeds []struct {
		URL string `json:"url"`
	} `json:"embeds"`
}

type NeynarPostResponse struct {
	Success bool `json:"success"`
	Cast    struct {
		Hash string `json:"hash"`
	} `json:"cast"`
}

type NeynarReactionResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Reaction struct {
		Hash string `json:"hash"`
	} `json:"reaction,omitempty"`
}

type NeynarFollowResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func NewFarcasterClient(creds *AccountCredentials) (*FarcasterClient, error) {
	if creds.APIKey == "" {
		return nil, errors.New("neynar API key required for Farcaster")
	}

	client := &FarcasterClient{
		creds:         creds,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		neynarAPIKey:  creds.APIKey,
		neynarBaseURL: "https://api.neynar.com/v2/farcaster",
		hubbleURL:     "https://hub.farcaster.standardcrypto.vc:2281", // Public hub
		authenticated: false,
	}

	// Parse signer key if provided
	if creds.PrivateKey != "" {
		keyBytes, err := hex.DecodeString(creds.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid signer private key: %w", err)
		}
		if len(keyBytes) == 64 {
			client.signerKey = ed25519.PrivateKey(keyBytes)
		} else if len(keyBytes) == 32 {
			client.signerKey = ed25519.NewKeyFromSeed(keyBytes)
		}
	}

	return client, nil
}

func (c *FarcasterClient) GetPlatformType() PlatformType {
	return PlatformFarcaster
}

func (c *FarcasterClient) Authenticate(ctx context.Context, credentials map[string]string) error {
	// Verify the signer UUID is valid by checking user info
	if c.creds.FID == 0 {
		return errors.New("FID required for authentication")
	}

	// Verify we can access the user
	_, err := c.GetProfile(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	c.authenticated = true
	return nil
}

func (c *FarcasterClient) IsAuthenticated() bool {
	return c.authenticated && c.creds.FID > 0
}

func (c *FarcasterClient) RefreshAuth(ctx context.Context) error {
	// Farcaster signers don't expire in traditional sense
	return nil
}

func (c *FarcasterClient) GetProfile(ctx context.Context) (*UserProfile, error) {
	url := fmt.Sprintf("%s/user?fid=%d", c.neynarBaseURL, c.creds.FID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		User NeynarUser `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &UserProfile{
		ID:          fmt.Sprintf("%d", result.User.FID),
		Username:    result.User.Username,
		DisplayName: result.User.DisplayName,
		AvatarURL:   result.User.PfpURL,
		Bio:         result.User.Bio,
		Followers:   result.User.FollowerCount,
		Following:   result.User.FollowingCount,
		Verified:    len(result.User.Verifications) > 0,
	}, nil
}

func (c *FarcasterClient) GetUserByUsername(ctx context.Context, username string) (*UserProfile, error) {
	url := fmt.Sprintf("%s/user/by_username?username=%s", c.neynarBaseURL, username)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUserNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		User NeynarUser `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &UserProfile{
		ID:          fmt.Sprintf("%d", result.User.FID),
		Username:    result.User.Username,
		DisplayName: result.User.DisplayName,
		AvatarURL:   result.User.PfpURL,
		Followers:   result.User.FollowerCount,
		Following:   result.User.FollowingCount,
	}, nil
}

func (c *FarcasterClient) Follow(ctx context.Context, targetFID string) (*ActionProof, error) {
	url := fmt.Sprintf("%s/user/follow", c.neynarBaseURL)
	
	payload := map[string]interface{}{
		"signer_uuid":  c.creds.AccessToken, // Neynar signer UUID
		"target_fids": []string{targetFID},
	}
	
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode == http.StatusConflict {
		return nil, ErrAlreadyFollowing
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("follow failed: %s", string(respBody))
	}

	var result NeynarFollowResponse
	json.Unmarshal(respBody, &result)

	return &ActionProof{
		Timestamp:   time.Now().Unix(),
		RawResponse: string(respBody),
		Metadata: map[string]string{
			"target_fid": targetFID,
			"action":     "follow",
		},
	}, nil
}

func (c *FarcasterClient) Unfollow(ctx context.Context, targetFID string) (*ActionProof, error) {
	url := fmt.Sprintf("%s/user/follow", c.neynarBaseURL)
	
	payload := map[string]interface{}{
		"signer_uuid": c.creds.AccessToken,
		"target_fids": []string{targetFID},
	}
	
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unfollow failed: %s", string(respBody))
	}

	return &ActionProof{
		Timestamp:   time.Now().Unix(),
		RawResponse: string(respBody),
		Metadata: map[string]string{
			"target_fid": targetFID,
			"action":     "unfollow",
		},
	}, nil
}

func (c *FarcasterClient) Like(ctx context.Context, castHash string) (*ActionProof, error) {
	url := fmt.Sprintf("%s/reaction", c.neynarBaseURL)
	
	payload := map[string]interface{}{
		"signer_uuid":   c.creds.AccessToken,
		"reaction_type": "like",
		"target":        castHash,
	}
	
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusConflict {
		return nil, ErrAlreadyLiked
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("like failed: %s", string(respBody))
	}

	var result NeynarReactionResponse
	json.Unmarshal(respBody, &result)

	return &ActionProof{
		CastHash:    castHash,
		PostURL:     fmt.Sprintf("https://warpcast.com/~/conversations/%s", castHash),
		Timestamp:   time.Now().Unix(),
		RawResponse: string(respBody),
		Metadata: map[string]string{
			"reaction_type": "like",
		},
	}, nil
}

func (c *FarcasterClient) Unlike(ctx context.Context, castHash string) (*ActionProof, error) {
	url := fmt.Sprintf("%s/reaction", c.neynarBaseURL)
	
	payload := map[string]interface{}{
		"signer_uuid":   c.creds.AccessToken,
		"reaction_type": "like",
		"target":        castHash,
	}
	
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unlike failed: %s", string(respBody))
	}

	return &ActionProof{
		CastHash:  castHash,
		Timestamp: time.Now().Unix(),
	}, nil
}

func (c *FarcasterClient) Repost(ctx context.Context, castHash string) (*ActionProof, error) {
	url := fmt.Sprintf("%s/reaction", c.neynarBaseURL)
	
	payload := map[string]interface{}{
		"signer_uuid":   c.creds.AccessToken,
		"reaction_type": "recast",
		"target":        castHash,
	}
	
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recast failed: %s", string(respBody))
	}

	return &ActionProof{
		CastHash:    castHash,
		PostURL:     fmt.Sprintf("https://warpcast.com/~/conversations/%s", castHash),
		Timestamp:   time.Now().Unix(),
		RawResponse: string(respBody),
		Metadata: map[string]string{
			"reaction_type": "recast",
		},
	}, nil
}

func (c *FarcasterClient) Post(ctx context.Context, content *PostContent) (*ActionProof, error) {
	url := fmt.Sprintf("%s/cast", c.neynarBaseURL)
	
	payload := map[string]interface{}{
		"signer_uuid": c.creds.AccessToken,
		"text":        content.Text,
	}

	if len(content.EmbedURLs) > 0 {
		embeds := make([]map[string]string, len(content.EmbedURLs))
		for i, u := range content.EmbedURLs {
			embeds[i] = map[string]string{"url": u}
		}
		payload["embeds"] = embeds
	}

	if content.ChannelID != "" {
		payload["channel_id"] = content.ChannelID
	}
	
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("post failed: %s", string(respBody))
	}

	var result NeynarPostResponse
	json.Unmarshal(respBody, &result)

	return &ActionProof{
		PostID:      result.Cast.Hash,
		CastHash:    result.Cast.Hash,
		PostURL:     fmt.Sprintf("https://warpcast.com/~/conversations/%s", result.Cast.Hash),
		Timestamp:   time.Now().Unix(),
		RawResponse: string(respBody),
	}, nil
}

func (c *FarcasterClient) Reply(ctx context.Context, parentHash string, content *PostContent) (*ActionProof, error) {
	url := fmt.Sprintf("%s/cast", c.neynarBaseURL)
	
	payload := map[string]interface{}{
		"signer_uuid": c.creds.AccessToken,
		"text":        content.Text,
		"parent":      parentHash,
	}

	if len(content.EmbedURLs) > 0 {
		embeds := make([]map[string]string, len(content.EmbedURLs))
		for i, u := range content.EmbedURLs {
			embeds[i] = map[string]string{"url": u}
		}
		payload["embeds"] = embeds
	}
	
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("reply failed: %s", string(respBody))
	}

	var result NeynarPostResponse
	json.Unmarshal(respBody, &result)

	return &ActionProof{
		PostID:      result.Cast.Hash,
		CastHash:    result.Cast.Hash,
		PostURL:     fmt.Sprintf("https://warpcast.com/~/conversations/%s", result.Cast.Hash),
		Timestamp:   time.Now().Unix(),
		RawResponse: string(respBody),
		Metadata: map[string]string{
			"parent_hash": parentHash,
		},
	}, nil
}

func (c *FarcasterClient) Quote(ctx context.Context, quotedHash string, content *PostContent) (*ActionProof, error) {
	// Farcaster quote = embed the original cast URL
	embedURL := fmt.Sprintf("https://warpcast.com/~/conversations/%s", quotedHash)
	content.EmbedURLs = append(content.EmbedURLs, embedURL)
	
	proof, err := c.Post(ctx, content)
	if err != nil {
		return nil, err
	}
	
	proof.Metadata = map[string]string{
		"quoted_hash": quotedHash,
	}
	return proof, nil
}

func (c *FarcasterClient) DeletePost(ctx context.Context, castHash string) error {
	url := fmt.Sprintf("%s/cast", c.neynarBaseURL)
	
	payload := map[string]interface{}{
		"signer_uuid": c.creds.AccessToken,
		"target_hash": castHash,
	}
	
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("api_key", c.neynarAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed: %s", string(respBody))
	}

	return nil
}

func (c *FarcasterClient) VerifyAction(ctx context.Context, actionType string, proof *ActionProof) (bool, error) {
	if proof.CastHash == "" {
		return false, errors.New("no cast hash in proof")
	}

	// Verify cast exists via Neynar
	url := fmt.Sprintf("%s/cast?identifier=%s&type=hash", c.neynarBaseURL, proof.CastHash)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("api_key", c.neynarAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (c *FarcasterClient) GetRateLimitStatus(ctx context.Context) (*RateLimitStatus, error) {
	// Neynar doesn't expose rate limits directly in responses
	// Return estimated limits based on their docs
	return &RateLimitStatus{
		Remaining: 100, // Conservative estimate
		Limit:     300, // Per minute
		ResetAt:   time.Now().Add(time.Minute).Unix(),
	}, nil
}
