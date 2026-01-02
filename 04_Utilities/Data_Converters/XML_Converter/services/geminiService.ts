import { GoogleGenerativeAI } from "@google/generative-ai";
import { AnalysisResult, ParsedMessage } from "../types";

export class GeminiService {
  private genAI: GoogleGenerativeAI;

  constructor() {
    const apiKey = import.meta.env.VITE_GEMINI_API_KEY || '';
    this.genAI = new GoogleGenerativeAI(apiKey);
  }

  async analyzeContext(messages: ParsedMessage[]): Promise<AnalysisResult> {
    const apiKey = import.meta.env.VITE_GEMINI_API_KEY;
    if (!apiKey) {
        throw new Error("Gemini API Key missing. Please add VITE_GEMINI_API_KEY to your .env file.");
    }

    // Prepare prompt data (reduce tokens by selecting key fields)
    const promptData = messages.map(m => ({
        date: m.date_sent || m.date,
        sender: m.contact_name || m.address,
        body: m.body ? m.body.substring(0, 200) : '[No Text]', // Truncate body to save tokens
        type: m.type === '1' ? 'Incoming' : 'Outgoing'
    }));

    const prompt = `
      Analyze the following sample of text messages from an XML export.
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
      }
    `;

    try {
      const model = this.genAI.getGenerativeModel({ 
          model: 'gemini-1.5-flash'
      });
      
      const result = await model.generateContent({
          contents: [{ role: 'user', parts: [{ text: prompt }] }],
          generationConfig: { responseMimeType: "application/json" }
      });
      
      const response = result.response;
      const text = response.text();
      
      if (!text) throw new Error("No response from AI");
      
      return JSON.parse(text) as AnalysisResult;
    } catch (error) {
      console.error("Gemini Analysis Failed:", error);
      throw error;
    }
  }
}