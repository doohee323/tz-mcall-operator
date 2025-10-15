/**
 * Authentication middleware for MCP Server
 * Supports API Key authentication via header or query parameter
 */

export interface AuthConfig {
  apiKeys: Set<string>;
  requireAuth: boolean;
}

export class AuthService {
  private config: AuthConfig;

  constructor() {
    const apiKeysEnv = process.env.MCP_API_KEYS || '';
    let apiKeys: string[] = [];
    
    try {
      // Try to parse as JSON array first
      if (apiKeysEnv.startsWith('[')) {
        apiKeys = JSON.parse(apiKeysEnv);
      } else {
        // Fallback to comma-separated values
        apiKeys = apiKeysEnv
          .split(',')
          .map(key => key.trim())
          .filter(key => key.length > 0);
      }
    } catch (error) {
      console.error('Error parsing MCP_API_KEYS:', error);
      // Fallback to comma-separated values
      apiKeys = apiKeysEnv
        .split(',')
        .map(key => key.trim())
        .filter(key => key.length > 0);
    }

    this.config = {
      apiKeys: new Set(apiKeys),
      requireAuth: process.env.MCP_REQUIRE_AUTH === 'true' || apiKeys.length > 0
    };

    if (this.config.requireAuth && this.config.apiKeys.size === 0) {
      console.warn('⚠️  MCP_REQUIRE_AUTH is true but no API keys configured!');
    }

    if (this.config.apiKeys.size > 0) {
      console.log(`🔐 API Key authentication enabled with ${this.config.apiKeys.size} key(s)`);
      console.log(`📋 Available API keys: ${Array.from(this.config.apiKeys).join(', ')}`);
    } else {
      console.log('🔓 API Key authentication disabled (open access)');
    }
  }

  /**
   * Extract API key from request
   */
  private extractApiKey(req: any): string | null {
    // 1. Check X-API-Key header
    const headerKey = req.headers['x-api-key'];
    if (headerKey) return headerKey;

    // 2. Check Authorization header (Bearer token style)
    const authHeader = req.headers['authorization'];
    if (authHeader?.startsWith('Bearer ')) {
      return authHeader.substring(7);
    }

    // 3. Check query parameter (less secure, but convenient for testing)
    const queryKey = req.query.apiKey || req.query.api_key;
    if (queryKey) return queryKey as string;

    return null;
  }

  /**
   * Validate API key
   */
  isValidApiKey(apiKey: string | null): boolean {
    if (!apiKey) return false;
    return this.config.apiKeys.has(apiKey);
  }

  /**
   * Express middleware for API key authentication
   */
  authenticate() {
    return (req: any, res: any, next: any) => {
      // Skip authentication if not required
      if (!this.config.requireAuth) {
        return next();
      }

      const apiKey = this.extractApiKey(req);

      if (!this.isValidApiKey(apiKey)) {
        console.log(`🚫 Unauthorized access attempt from ${req.ip}`);
        console.log(`🔍 Received API key: "${apiKey}"`);
        console.log(`📋 Valid API keys: ${Array.from(this.config.apiKeys).join(', ')}`);
        return res.status(401).json({
          error: 'Unauthorized',
          message: 'Valid API key required. Provide via X-API-Key header, Authorization: Bearer token, or apiKey query parameter.'
        });
      }

      // Log successful authentication
      console.log(`✅ Authenticated request from ${req.ip}`);
      next();
    };
  }

  /**
   * Optional: Rate limiting per API key
   */
  private requestCounts = new Map<string, { count: number; resetTime: number }>();

  rateLimit(maxRequests: number = 100, windowMs: number = 60000) {
    return (req: any, res: any, next: any) => {
      const apiKey = this.extractApiKey(req);
      if (!apiKey) return next();

      const now = Date.now();
      const key = apiKey;
      const record = this.requestCounts.get(key);

      if (!record || now > record.resetTime) {
        this.requestCounts.set(key, { count: 1, resetTime: now + windowMs });
        return next();
      }

      if (record.count >= maxRequests) {
        return res.status(429).json({
          error: 'Too Many Requests',
          message: `Rate limit exceeded. Max ${maxRequests} requests per ${windowMs / 1000}s.`
        });
      }

      record.count++;
      next();
    };
  }
}

