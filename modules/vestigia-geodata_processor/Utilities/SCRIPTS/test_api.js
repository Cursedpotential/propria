async function test() {
  try {
    console.log("Testing API...");
    const response = await fetch('http://localhost:3001/api/ask-agent', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({agent: 'gemini-cli', prompt: 'Please reply with the single word: Success'})
    });
    const json = await response.json();
    console.log('Result:', json);
  } catch (e) {
    console.error('Error:', e);
  }
}
test();