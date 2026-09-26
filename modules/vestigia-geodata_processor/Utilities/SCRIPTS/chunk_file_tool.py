#!/usr/bin/env python3
"""
Large File Chunker Tool

This tool streams large files and creates chunks based on specified bracket pairs,
ensuring clean separation of topics in the data.
"""

import os
import sys
from pathlib import Path
import argparse
import logging


class FileChunker:
    """
    A class to handle chunking large files between specified bracket pairs.
    """
    
    def __init__(self, file_path, left_bracket="{", right_bracket="}", chunk_dir="chunks"):
        self.file_path = Path(file_path)
        self.left_bracket = left_bracket
        self.right_bracket = right_bracket
        self.chunk_dir = Path(chunk_dir)
        
        # Validate input file exists
        if not self.file_path.exists():
            raise FileNotFoundError(f"Input file does not exist: {self.file_path}")
    
    def create_chunk_directory(self):
        """Create the directory for storing chunks."""
        self.chunk_dir.mkdir(exist_ok=True)
        logging.info(f"Created chunk directory: {self.chunk_dir.absolute()}")
    
    def chunk_file(self, buffer_size=8192):
        """
        Stream the file and create chunks between bracket pairs.

        Args:
            buffer_size: Size of buffer to read at a time for large file handling

        Returns:
            int: Number of chunks created
        """
        chunk_count = 0
        current_chunk = ""
        bracket_balance = 0
        inside_brackets = False

        # Keep track of leftover characters that might form brackets across boundaries
        boundary_buffer = ""

        with open(self.file_path, 'r', encoding='utf-8') as file:
            while True:
                # Read the next buffer of data
                raw_chunk = file.read(buffer_size)
                if not raw_chunk:  # End of file
                    # Process any remaining content in the boundary buffer
                    if boundary_buffer:
                        for char in boundary_buffer:
                            if char == self.left_bracket:
                                if not inside_brackets:
                                    # Starting a new chunk
                                    inside_brackets = True
                                    bracket_balance = 1
                                    current_chunk = char
                                else:
                                    # Nested bracket
                                    bracket_balance += 1
                                    current_chunk += char
                            elif char == self.right_bracket:
                                if inside_brackets:
                                    bracket_balance -= 1
                                    current_chunk += char

                                    # If we've closed all open brackets in this chunk
                                    if bracket_balance == 0:
                                        # Save the chunk
                                        chunk_count += 1
                                        chunk_filename = self.chunk_dir / f"chunk_{chunk_count:04d}.txt"

                                        with open(chunk_filename, 'w', encoding='utf-8') as chunk_file:
                                            chunk_file.write(current_chunk)

                                        logging.info(f"Saved chunk {chunk_count}: {chunk_filename.name}")

                                        # Reset for next chunk
                                        current_chunk = ""
                                        inside_brackets = False
                            else:
                                # Regular character
                                if inside_brackets:
                                    current_chunk += char
                    break

                # Combine the boundary buffer with the new chunk
                chunk = boundary_buffer + raw_chunk
                boundary_buffer = ""

                # Process character by character
                i = 0
                while i < len(chunk):
                    char = chunk[i]

                    if char == self.left_bracket:
                        if not inside_brackets:
                            # Starting a new chunk
                            inside_brackets = True
                            bracket_balance = 1
                            current_chunk = char
                        else:
                            # Nested bracket
                            bracket_balance += 1
                            current_chunk += char
                    elif char == self.right_bracket:
                        if inside_brackets:
                            bracket_balance -= 1
                            current_chunk += char

                            # If we've closed all open brackets in this chunk
                            if bracket_balance == 0:
                                # Save the chunk
                                chunk_count += 1
                                chunk_filename = self.chunk_dir / f"chunk_{chunk_count:04d}.txt"

                                with open(chunk_filename, 'w', encoding='utf-8') as chunk_file:
                                    chunk_file.write(current_chunk)

                                logging.info(f"Saved chunk {chunk_count}: {chunk_filename.name}")

                                # Reset for next chunk
                                current_chunk = ""
                                inside_brackets = False
                    else:
                        # Regular character
                        if inside_brackets:
                            current_chunk += char

                    i += 1

                # Keep the last few characters in the boundary buffer to handle cases
                # where brackets might be split across buffer boundaries
                if len(chunk) >= max(len(self.left_bracket), len(self.right_bracket)):
                    boundary_chars_needed = max(len(self.left_bracket), len(self.right_bracket)) - 1
                    if boundary_chars_needed > 0:
                        boundary_buffer = chunk[-boundary_chars_needed:]

                        # Adjust current_chunk to remove the boundary buffer content if we're inside brackets
                        if inside_brackets and len(current_chunk) >= boundary_chars_needed:
                            current_chunk = current_chunk[:-boundary_chars_needed]

        # If we reach end of file and still have a chunk in progress
        if current_chunk and inside_brackets:
            # If we're still inside brackets when file ends, save the incomplete chunk
            chunk_count += 1
            chunk_filename = self.chunk_dir / f"chunk_incomplete_{chunk_count:04d}.txt"

            with open(chunk_filename, 'w', encoding='utf-8') as chunk_file:
                chunk_file.write(current_chunk)

            logging.warning(f"Saved incomplete chunk (did not close bracket): {chunk_filename.name}")

        return chunk_count
    
    def get_bracket_count(self, text, left_bracket, right_bracket):
        """Helper function to count open/close brackets in a text."""
        left_count = text.count(left_bracket)
        right_count = text.count(right_bracket)
        return left_count, right_count


def main():
    parser = argparse.ArgumentParser(description="Chunk large files between specified bracket pairs")
    parser.add_argument("input_file", help="Path to the input file to chunk")
    parser.add_argument("--left-bracket", "-l", default="{", help="Left bracket character (default: {)")
    parser.add_argument("--right-bracket", "-r", default="}", help="Right bracket character (default: })")
    parser.add_argument("--chunk-dir", "-d", default="chunks", help="Directory to store chunks (default: chunks)")
    parser.add_argument("--buffer-size", "-b", type=int, default=8192, help="Buffer size for reading large files (default: 8192)")
    parser.add_argument("--verbose", "-v", action="store_true", help="Enable verbose logging")
    
    args = parser.parse_args()
    
    # Setup logging
    level = logging.DEBUG if args.verbose else logging.INFO
    logging.basicConfig(level=level, format='%(asctime)s - %(levelname)s - %(message)s')
    
    try:
        chunker = FileChunker(
            file_path=args.input_file,
            left_bracket=args.left_bracket,
            right_bracket=args.right_bracket,
            chunk_dir=args.chunk_dir
        )
        
        logging.info(f"Starting to chunk file: {args.input_file}")
        logging.info(f"Using brackets: '{args.left_bracket}' and '{args.right_bracket}'")
        
        # Create chunk directory
        chunker.create_chunk_directory()
        
        # Process the file
        chunked_count = chunker.chunk_file(buffer_size=args.buffer_size)
        
        logging.info(f"Successfully processed {chunked_count} chunks from {args.input_file}")
        
    except Exception as e:
        logging.error(f"Error processing file: {str(e)}")
        sys.exit(1)


if __name__ == "__main__":
    main()