import express from 'express';
import cors from 'cors';
import bodyParser from 'body-parser';
import { exec } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = 3001;

app.use(cors());
app.use(bodyParser.json());

const runCommand = (command) => {
    return new Promise((resolve) => {
        console.log(`Executing: ${command}`);
        exec(command, { maxBuffer: 1024 * 1024 * 10, cwd: process.cwd() }, (error, stdout, stderr) => {
            if (error) {
                console.warn(`Command output/stderr: ${stderr}`);
            }
            resolve(stdout || stderr); 
        });
    });
};

app.post('/api/ask-agent', async (req, res) => {
    const { agent, prompt } = req.body;
    const promptFile = path.join(__dirname, 'agent_prompt.txt');
    fs.writeFileSync(promptFile, prompt, 'utf8');

    try {
        let cmd = '';
        switch (agent) {
            case 'claude': 
                cmd = `type "${promptFile}" | "C:\\Users\\matts\\.local\\bin\\claude.exe"`; 
                break;
            case 'gemini-cli': 
                cmd = `type "${promptFile}" | "C:\\Users\\matts\\AppData\\Roaming\\npm\\gemini.cmd"`; 
                break;
            case 'qwen': 
                cmd = `type "${promptFile}" | "C:\\Users\\matts\\AppData\\Roaming\\npm\\qwen.cmd"`; 
                break;
            default: 
                throw new Error(`Unknown agent: ${agent}`);
        }

        const result = await runCommand(cmd);
        res.json({ success: true, data: result });
    } catch (error) {
        res.status(500).json({ success: false, error: error.message });
    }
});

app.listen(PORT, () => {
    console.log(`Agent Gateway running on http://localhost:${PORT}`);
});