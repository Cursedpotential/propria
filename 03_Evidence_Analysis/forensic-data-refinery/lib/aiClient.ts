
import { GoogleGenAI, GenerateContentResponse, Type } from '@google/genai';
import { AppSettings } from '../types.ts';

const analysisSchema = {
  type: Type.OBJECT,
  properties: {
    entities: {
      type: Type.ARRAY,
      description: "A list of named entities found in the text, such as people, organizations, locations, or specific assets (e.g., usernames, phone numbers).",
      items: {
        type: Type.OBJECT,
        properties: {
          name: { type: Type.STRING, description: "The entity's name." },
          type: { type: Type.STRING, description: "The type of entity (e.g., 'Person', 'Location', 'Asset')." }
        },
        required: ["name", "type"],
      }
    },
    timeline_date: {
      type: Type.STRING,
      description: "The most relevant date or timestamp for this text chunk in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ). If no date is apparent, return null."
    },
    summary: {
        type: Type.STRING,
        description: "A one-sentence summary of the text chunk."
    }
  },
  required: ["entities", "timeline_date", "summary"],
};

export class ForensicAI {
  private settings: AppSettings;
  private apiKey: string | null = null;

  constructor(settings: AppSettings) {
    this.settings = settings;
    this.apiKey = settings.geminiApiKey;
  }

  async analyzeChunk(text: string): Promise<any> {
    if (!this.apiKey) {
      throw new Error("Gemini API key is not configured in Settings. Analysis is not available.");
    }

    const prompt = `
      Analyze the following text from a chat log for forensic purposes.
      Extract key entities (people, locations, assets), determine the most relevant date (ISO 8601 format or null), and provide a one-sentence summary.
      Respond ONLY with a valid JSON object that conforms to the requested schema.

      Text to analyze:
      ---
      ${text}
      ---
    `;

    if (this.settings.useProxy && this.settings.proxyUrl) {
      // Proxy path: relies on prompt engineering + response_mime_type for JSON.
      const payloadForWorker = {
          model: 'gemini-2.5-flash',
          contents: [{ role: 'user', parts: [{ text: prompt }] }],
          generationConfig: {
              response_mime_type: 'application/json',
              temperature: 0.1,
          }
      };
      
      if (this.settings.debugMode) {
          console.log("--- AI Proxy Request Payload ---", JSON.stringify(payloadForWorker, null, 2));
      }
      return this.analyzeViaProxy(payloadForWorker);
    } else {
      // Direct path: uses SDK with schema enforcement for higher reliability.
      const ai = new GoogleGenAI({ apiKey: this.apiKey, vertexai: true });
      const sdkPayload = {
          model: 'gemini-2.5-flash',
          contents: [{ role: 'user', parts: [{ text: prompt }] }],
          config: {
              responseMimeType: 'application/json',
              responseSchema: analysisSchema,
              temperature: 0.1,
          },
      };
      if (this.settings.debugMode) {
          console.log("--- AI Direct Request Payload ---", JSON.stringify(sdkPayload, null, 2));
      }
      return this.analyzeDirectly(ai, sdkPayload);
    }
  }

  private async analyzeDirectly(ai: GoogleGenAI, payload: any): Promise<any> {
    try {
      const result: GenerateContentResponse = await ai.models.generateContent(payload);
      
      if (this.settings.debugMode) {
        console.log("--- AI Direct Response ---", result);
      }

      const responseText = result.text;
      return JSON.parse(responseText);
    } catch (error) {
      console.error("Direct AI analysis failed:", error);
      const errorMessage = error instanceof Error ? error.message : String(error);
      throw new Error(`Direct Gemini API Error: ${errorMessage}`);
    }
  }

  private async analyzeViaProxy(payload: any): Promise<any> {
    try {
      const response = await fetch(this.settings.proxyUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Auth': this.apiKey!,
        },
        body: JSON.stringify(payload),
      });

      if (this.settings.debugMode) {
        const headers: Record<string, string> = {};
        response.headers.forEach((value, key) => {
          headers[key] = value;
        });
        console.log("--- AI Proxy Response Headers ---", headers);
      }

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Proxy request failed with status ${response.status}: ${errorText}`);
      }

      const jsonResponse = await response.json();

      if (this.settings.debugMode) {
        console.log("--- AI Proxy Raw JSON Response ---", JSON.stringify(jsonResponse, null, 2));
      }

      const responseText = jsonResponse?.candidates?.[0]?.content?.parts?.[0]?.text;
      
      if (!responseText) {
          const errorDetails = jsonResponse?.error?.message || JSON.stringify(jsonResponse);
          console.error("Invalid response structure from proxy or API error:", jsonResponse);
          throw new Error(`Invalid response from proxy: ${errorDetails}`);
      }

      return JSON.parse(responseText);

    } catch (error) {
      console.error("Proxy AI analysis failed:", error);
      const errorMessage = error instanceof Error ? error.message : String(error);
      throw new Error(`Proxy AI analysis failed: ${errorMessage}`);
    }
  }
}
