import { AnalysisResult, ParsedMessage } from "../types";

export type AIProvider = 'openrouter' | 'gemini' | 'anthropic' | 'openai';

interface AIConfig {
  provider: AIProvider;
  apiKey: string;
  model?: string;
  baseUrl?: string;
}

export class AIService {
  private config: AIConfig;

  constructor(config?: Partial<AIConfig>) {
    // Auto-detect provider from environment variables
    const provider = config?.provider ||
      (import.meta.env.VITE_OPENROUTER_API_KEY ? 'openrouter' :
       import.meta.env.VITE_GEMINI_API_KEY ? 'gemini' :
       import.meta.env.VITE_ANTHROPIC_API_KEY ? 'anthropic' :
       import.meta.env.VITE_OPENAI_API_KEY ? 'openai' : 'openrouter');

    const apiKey = config?.apiKey ||
      import.meta.env[`VITE_${provider.toUpperCase()}_API_KEY`] || '';

    // Default models for each provider
    const defaultModels: Record<AIProvider, string> = {
      openrouter: 'anthropic/claude-3.5-sonnet',
      gemini: 'gemini-1.5-flash',
      anthropic: 'claude-3-5-sonnet-20241022',
      openai: 'gpt-4o'
    };

    this.config = {
      provider,
      apiKey,
      model: config?.model || defaultModels[provider],
      baseUrl: config?.baseUrl || (provider === 'openrouter' ? 'https://openrouter.ai/api/v1' : undefined)
    };
  }

  async analyzeContext(messages: ParsedMessage[]): Promise<AnalysisResult> {
    if (!this.config.apiKey) {
      throw new Error(`${this.config.provider.toUpperCase()} API Key missing. Please add VITE_${this.config.provider.toUpperCase()}_API_KEY to your .env file.`);
    }

    // Prepare prompt data
    const promptData = messages.map(m => ({
      date: m.date_sent || m.date,
      sender: m.contact_name || m.address,
      body: m.body ? m.body.substring(0, 200) : '[No Text]',
      type: m.type === '1' ? 'Incoming' : 'Outgoing'
    }));

    const systemPrompt = `You are an expert at analyzing text message conversations. Provide structured analysis in JSON format.`;

    const userPrompt = `Analyze the following sample of text messages from an XML export.
Provide a structured summary containing:
1. A short paragraph summary of the conversation tone and context.
2. A list of likely participants (names or numbers).
3. The date range covered in this sample.
4. Key topics discussed.

Data:
${JSON.stringify(promptData, null, 2)}

Output JSON format:
{
  "summary": "string",
  "participants": ["string"],
  "dateRange": "string",
  "topics": ["string"]
}`;

    switch (this.config.provider) {
      case 'openrouter':
        return await this.callOpenRouter(systemPrompt, userPrompt);
      case 'gemini':
        return await this.callGemini(userPrompt);
      case 'anthropic':
        return await this.callAnthropic(systemPrompt, userPrompt);
      case 'openai':
        return await this.callOpenAI(systemPrompt, userPrompt);
      default:
        throw new Error(`Unsupported AI provider: ${this.config.provider}`);
    }
  }

  private async callOpenRouter(systemPrompt: string, userPrompt: string): Promise<AnalysisResult> {
    const response = await fetch(`${this.config.baseUrl}/chat/completions`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.config.apiKey}`,
        'Content-Type': 'application/json',
        'HTTP-Referer': window.location.origin,
        'X-Title': 'XML ETL Studio'
      },
      body: JSON.stringify({
        model: this.config.model,
        messages: [
          { role: 'system', content: systemPrompt },
          { role: 'user', content: userPrompt }
        ],
        response_format: { type: 'json_object' },
        temperature: 0.7
      })
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`OpenRouter API error: ${error}`);
    }

    const data = await response.json();
    const content = data.choices[0]?.message?.content;

    if (!content) throw new Error("No response from OpenRouter");

    return JSON.parse(content) as AnalysisResult;
  }

  private async callGemini(userPrompt: string): Promise<AnalysisResult> {
    // Dynamic import to avoid loading if not needed
    const { GoogleGenerativeAI } = await import("@google/generative-ai");
    const genAI = new GoogleGenerativeAI(this.config.apiKey);

    const model = genAI.getGenerativeModel({
      model: this.config.model!
    });

    const result = await model.generateContent({
      contents: [{ role: 'user', parts: [{ text: userPrompt }] }],
      generationConfig: { responseMimeType: "application/json" }
    });

    const response = result.response;
    const text = response.text();

    if (!text) throw new Error("No response from Gemini");

    return JSON.parse(text) as AnalysisResult;
  }

  private async callAnthropic(systemPrompt: string, userPrompt: string): Promise<AnalysisResult> {
    const response = await fetch('https://api.anthropic.com/v1/messages', {
      method: 'POST',
      headers: {
        'x-api-key': this.config.apiKey,
        'anthropic-version': '2023-06-01',
        'content-type': 'application/json'
      },
      body: JSON.stringify({
        model: this.config.model,
        max_tokens: 1024,
        system: systemPrompt,
        messages: [
          { role: 'user', content: userPrompt }
        ]
      })
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Anthropic API error: ${error}`);
    }

    const data = await response.json();
    const content = data.content[0]?.text;

    if (!content) throw new Error("No response from Anthropic");

    // Anthropic doesn't guarantee JSON mode, so we need to extract it
    const jsonMatch = content.match(/\{[\s\S]*\}/);
    if (!jsonMatch) throw new Error("Invalid JSON response from Anthropic");

    return JSON.parse(jsonMatch[0]) as AnalysisResult;
  }

  private async callOpenAI(systemPrompt: string, userPrompt: string): Promise<AnalysisResult> {
    const response = await fetch('https://api.openai.com/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.config.apiKey}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        model: this.config.model,
        messages: [
          { role: 'system', content: systemPrompt },
          { role: 'user', content: userPrompt }
        ],
        response_format: { type: 'json_object' },
        temperature: 0.7
      })
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`OpenAI API error: ${error}`);
    }

    const data = await response.json();
    const content = data.choices[0]?.message?.content;

    if (!content) throw new Error("No response from OpenAI");

    return JSON.parse(content) as AnalysisResult;
  }

  getProviderName(): string {
    return this.config.provider.charAt(0).toUpperCase() + this.config.provider.slice(1);
  }

  getModel(): string {
    return this.config.model || 'unknown';
  }
}
