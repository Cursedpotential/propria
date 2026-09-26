import os
import sys
import argparse
import hashlib
import logging
import json
from datetime import datetime
import difflib
import heapq
import re
import math
from collections import Counter

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

# Constants
MAX_SUMMARY_TOKENS = 200
HEAD_TAIL_LINES = 10
LARGE_FILE_THRESHOLD = 10 * 1024 * 1024  # 10 MB limit for full NLP analysis to prevent OOM
CHUNK_SIZE = 65536 # 64kb for hashing

try:
    import nltk
    from nltk.tokenize import sent_tokenize, word_tokenize
    from nltk.corpus import stopwords
    # Ensure resources are downloaded (quietly)
    try:
        nltk.data.find('tokenizers/punkt')
    except LookupError:
        nltk.download('punkt', quiet=True)
    try:
        nltk.data.find('tokenizers/punkt_tab')
    except LookupError:
        nltk.download('punkt_tab', quiet=True)
    try:
        nltk.data.find('corpora/stopwords')
    except LookupError:
        nltk.download('stopwords', quiet=True)
except ImportError:
    nltk = None

try:
    from sklearn.feature_extraction.text import TfidfVectorizer
    from sklearn.metrics.pairwise import cosine_similarity
except ImportError:
    TfidfVectorizer = None

def get_file_metadata(filepath):
    """Get basic file metadata."""
    try:
        stat = os.stat(filepath)
        return {
            "path": filepath,
            "size": stat.st_size,
            "created": datetime.fromtimestamp(stat.st_ctime).isoformat(),
            "modified": datetime.fromtimestamp(stat.st_mtime).isoformat()
        }
    except Exception as e:
        return {"error": str(e)}

def read_head_tail(filepath, lines=HEAD_TAIL_LINES):
    """Reads the first and last N lines of a file."""
    head = []
    tail = []
    try:
        with open(filepath, 'r', encoding='utf-8', errors='replace') as f:
            # Read Head
            for _ in range(lines):
                line = f.readline()
                if not line:
                    break
                head.append(line.strip())
            
            # Read Tail (efficiently using seek if possible, but for simplicity/universality read all lines if small, or seek if large)
            # For strict correctness on varied encoding/newlines, reading all is safest but slow for huge files.
            # Optimization: Seek to end and read backwards chunk by chunk.
            
            if os.path.getsize(filepath) > 1024 * 1024: # > 1MB use seek method
                f.seek(0, os.SEEK_END)
                file_size = f.tell()
                block_size = 1024
                data = ""
                pos = file_size
                while pos > 0 and len(data.splitlines()) <= lines + 1:
                    read_size = min(block_size, pos)
                    pos -= read_size
                    f.seek(pos)
                    data = f.read(read_size) + data
                tail = data.splitlines()[-lines:]
            else:
                # Small file, just read the rest
                remaining = f.readlines()
                all_lines = head + [x.strip() for x in remaining]
                if len(all_lines) > lines:
                    tail = all_lines[-lines:]
                else:
                    tail = [] # Overlap handled by logic
    except Exception as e:
        return [], [], str(e)
    
    return head, tail, None

def generate_summary(text):
    """Generates a summary using basic frequency analysis (TextRank-like)."""
    if not text or not nltk:
        return "NLTK not available or no text content."
    
    sentences = sent_tokenize(text)
    if len(sentences) <= 5:
        return text # Short enough
    
    stop_words = set(stopwords.words('english'))
    word_frequencies = {}
    
    words = word_tokenize(text.lower())
    for word in words:
        if word.isalnum() and word not in stop_words:
            word_frequencies[word] = word_frequencies.get(word, 0) + 1
            
    if not word_frequencies:
        return "No significant words found."

    max_freq = max(word_frequencies.values())
    for word in word_frequencies:
        word_frequencies[word] = word_frequencies[word] / max_freq
        
    sentence_scores = {}
    for sent in sentences:
        for word in word_tokenize(sent.lower()):
            if word in word_frequencies:
                if len(sent.split(' ')) < 30: # limit sentence length
                    sentence_scores[sent] = sentence_scores.get(sent, 0) + word_frequencies[word]

    summary_sentences = heapq.nlargest(3, sentence_scores, key=sentence_scores.get)
    return ' '.join(summary_sentences)

def calculate_file_hash(filepath):
    """Calculates MD5 hash of a file."""
    hasher = hashlib.md5()
    try:
        with open(filepath, 'rb') as f:
            for chunk in iter(lambda: f.read(CHUNK_SIZE), b""):
                hasher.update(chunk)
        return hasher.hexdigest()
    except Exception as e:
        return None

def analyze_file(filepath):
    """Analyzes a single file."""
    meta = get_file_metadata(filepath)
    if "error" in meta:
        return meta

    head, tail, err = read_head_tail(filepath)
    meta['head'] = head
    meta['tail'] = tail
    
    if err:
        meta['read_error'] = err
        return meta

    # Summarization (Only if text and not too huge)
    if meta['size'] < LARGE_FILE_THRESHOLD:
        try:
            with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
                content = f.read()
            meta['summary'] = generate_summary(content)
        except Exception as e:
            meta['summary_error'] = str(e)
    else:
        meta['summary'] = "File too large for full content NLP analysis."

    return meta

def find_duplicates(file_paths):
    """Finds exact duplicates based on hash."""
    hashes = {}
    duplicates = []
    
    for path in file_paths:
        if not os.path.exists(path):
            continue
        file_hash = calculate_file_hash(path)
        if not file_hash:
            continue
            
        if file_hash in hashes:
            duplicates.append({
                "original": hashes[file_hash],
                "duplicate": path,
                "hash": file_hash
            })
        else:
            hashes[file_hash] = path
            
    return duplicates

def check_similarity(file_paths):
    """Checks similarity using TF-IDF and Cosine Similarity."""
    if not TfidfVectorizer:
        return {"error": "scikit-learn not installed."}
    
    documents = []
    valid_paths = []
    
    for path in file_paths:
        if not os.path.exists(path) or os.path.getsize(path) > LARGE_FILE_THRESHOLD:
            continue
        try:
            with open(path, 'r', encoding='utf-8', errors='ignore') as f:
                documents.append(f.read())
            valid_paths.append(path)
        except:
            pass
            
    if len(documents) < 2:
        return {"message": "Not enough valid text files to compare."}
    
    tfidf_vectorizer = TfidfVectorizer()
    tfidf_matrix = tfidf_vectorizer.fit_transform(documents)
    cosine_sim = cosine_similarity(tfidf_matrix, tfidf_matrix)
    
    results = []
    for i in range(len(valid_paths)):
        for j in range(i + 1, len(valid_paths)):
            score = cosine_sim[i][j]
            if score > 0.5: # Threshold
                results.append({
                    "file1": valid_paths[i],
                    "file2": valid_paths[j],
                    "similarity_score": round(score, 4)
                })
                
    return sorted(results, key=lambda x: x['similarity_score'], reverse=True)

def main():
    parser = argparse.ArgumentParser(description="File Analyzer Tool")
    subparsers = parser.add_subparsers(dest="command")

    # Analyze Command
    analyze_parser = subparsers.add_parser("analyze", help="Analyze a specific file")
    analyze_parser.add_argument("filepath", help="Path to the file")

    # Dedupe Command
    dedupe_parser = subparsers.add_parser("dedupe", help="Find exact duplicates in a list of files")
    dedupe_parser.add_argument("files", nargs="+", help="List of file paths")

    # Similarity Command
    sim_parser = subparsers.add_parser("similarity", help="Check similarity between files")
    sim_parser.add_argument("files", nargs="+", help="List of file paths")

    args = parser.parse_args()

    if args.command == "analyze":
        print(json.dumps(analyze_file(args.filepath), indent=2))
    elif args.command == "dedupe":
        print(json.dumps(find_duplicates(args.files), indent=2))
    elif args.command == "similarity":
        print(json.dumps(check_similarity(args.files), indent=2))
    else:
        parser.print_help()

if __name__ == "__main__":
    main()
