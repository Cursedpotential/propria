
/**
 * Cloudflare Worker to proxy requests to the Google Gemini API.
 *
 * This worker expects a POST request with a JSON body. It requires an 'X-Auth' 
 * header containing the Gemini API key.
 * The request body from the client should include a 'model' property, which this
 * worker uses to construct the correct API URL. The 'model' property is then
 * removed from the payload before forwarding it to Google's API.
 */

export default {
  async fetch(request, env, ctx) {
    // Handle CORS preflight requests
    if (request.method === 'OPTIONS') {
      return new Response(null, {
        headers: {
          'Access-Control-Allow-Origin': '*',
          'Access-Control-Allow-Methods': 'POST, OPTIONS',
          'Access-Control-Allow-Headers': 'Content-Type, X-Auth',
        },
      });
    }
    
    if (request.method !== 'POST') {
      return new Response('Expected POST', { status: 405 });
    }

    const apiKey = request.headers.get('X-Auth');
    if (!apiKey) {
      return new Response('Missing X-Auth header', { status: 401 });
    }

    const clientRequestBody = await request.json();
    
    // Destructure the model from the body, the rest is the payload for Google.
    const { model, ...googleApiBody } = clientRequestBody;

    if (!model) {
        return new Response('Missing "model" in request body', { status: 400 });
    }

    // Use the public REST endpoint. The client must send a REST-compatible payload.
    const GOOGLE_API_URL = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent`;

    const googleApiRequest = new Request(GOOGLE_API_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-goog-api-key': apiKey,
      },
      // Forward the body *without* the model property.
      body: JSON.stringify(googleApiBody),
    });

    try {
      const googleApiResponse = await fetch(googleApiRequest);

      // Create a new response, copying the headers and body from Google's response.
      const response = new Response(googleApiResponse.body, googleApiResponse);
      
      // Set CORS headers for the actual response
      response.headers.set('Access-Control-Allow-Origin', '*');

      return response;

    } catch (error) {
      console.error('Error forwarding request to Google API:', error);
      return new Response('Error proxying request to Gemini API', { status: 500 });
    }
  },
};
