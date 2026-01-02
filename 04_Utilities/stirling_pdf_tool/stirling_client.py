import argparse
import requests
import os
import sys

BASE_URL = "http://localhost:8080/api/v1"

def check_health():
    try:
        # Stirling-PDF doesn't have a standard /health endpoint documented in all versions, 
        # but hitting the root API or a simple endpoint is a good check.
        # We'll try to list something simple or just connect.
        response = requests.get("http://localhost:8080/swagger-ui/index.html")
        if response.status_code == 200:
            print("Status: Online")
            return True
        else:
            print(f"Status: Issues detected (Code: {response.status_code})")
            return False
    except Exception as e:
        print(f"Status: Offline ({e})")
        return False

def merge_pdfs(output_path, input_paths):
    url = f"{BASE_URL}/general/merge"
    files = []
    try:
        for path in input_paths:
            if not os.path.exists(path):
                print(f"Error: File not found - {path}")
                return
            files.append(('fileInput', (os.path.basename(path), open(path, 'rb'), 'application/pdf')))

        response = requests.post(url, files=files)
        
        if response.status_code == 200:
            with open(output_path, 'wb') as f:
                f.write(response.content)
            print(f"Successfully merged {len(input_paths)} files to {output_path}")
        else:
            print(f"Error merging files: {response.status_code} - {response.text}")
    except Exception as e:
        print(f"Exception: {e}")
    finally:
        for _, file_tuple in files:
            file_tuple[1].close()

def split_pdf(input_path, output_dir):
    # This endpoint typically returns a zip or individual files depending on implementation.
    # For simplicity in this script, we'll assume standard split behavior.
    # Note: Stirling API for split might be /general/split-pages or similar.
    # We will use the 'split-pages' endpoint which creates a zip of split files usually.
    url = f"{BASE_URL}/general/split-pages"
    
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)

    try:
        with open(input_path, 'rb') as f:
            files = {'fileInput': (os.path.basename(input_path), f, 'application/pdf')}
            # Default to splitting all pages
            data = {'pageNumbers': 'all'} 
            response = requests.post(url, files=files, data=data)

        if response.status_code == 200:
            # If it's a zip file (common for multiple outputs)
            output_zip = os.path.join(output_dir, "split_result.zip")
            with open(output_zip, 'wb') as f:
                f.write(response.content)
            print(f"Split successful. Downloaded to {output_zip}")
            print("Note: You may need to unzip this file.")
        else:
            print(f"Error splitting file: {response.status_code} - {response.text}")

    except Exception as e:
        print(f"Exception: {e}")

def convert_to_pdf(input_path, output_path):
    url = f"{BASE_URL}/convert/file/pdf"
    try:
        with open(input_path, 'rb') as f:
            files = {'fileInput': (os.path.basename(input_path), f, 'application/octet-stream')}
            response = requests.post(url, files=files)

        if response.status_code == 200:
            with open(output_path, 'wb') as f:
                f.write(response.content)
            print(f"Successfully converted {input_path} to {output_path}")
        else:
            print(f"Error converting file: {response.status_code} - {response.text}")
    except Exception as e:
        print(f"Exception: {e}")

def main():
    parser = argparse.ArgumentParser(description="Stirling-PDF CLI Tool")
    subparsers = parser.add_subparsers(dest="command", help="Available commands")

    # Health command
    subparsers.add_parser("health", help="Check if Stirling-PDF is running")

    # Merge command
    merge_parser = subparsers.add_parser("merge", help="Merge multiple PDFs")
    merge_parser.add_argument("output", help="Path to output PDF")
    merge_parser.add_argument("inputs", nargs="+", help="Paths to input PDFs")

    # Split command
    split_parser = subparsers.add_parser("split", help="Split a PDF")
    split_parser.add_argument("input", help="Path to input PDF")
    split_parser.add_argument("output_dir", help="Directory to save output")

    # Convert command
    convert_parser = subparsers.add_parser("convert", help="Convert a file to PDF")
    convert_parser.add_argument("input", help="Path to input file")
    convert_parser.add_argument("output", help="Path to output PDF")

    args = parser.parse_args()

    if args.command == "health":
        check_health()
    elif args.command == "merge":
        merge_pdfs(args.output, args.inputs)
    elif args.command == "split":
        split_pdf(args.input, args.output_dir)
    elif args.command == "convert":
        convert_to_pdf(args.input, args.output)
    else:
        parser.print_help()

if __name__ == "__main__":
    main()
