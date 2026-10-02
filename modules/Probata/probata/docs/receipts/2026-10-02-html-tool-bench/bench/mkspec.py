import json,sys
py=sys.argv[1]; kind=sys.argv[2]
g=["webbed_html_extract_text","webbed_read_html_blocks","selectolax_lexbor","lxml","beautifulsoup4","html2text","markitdown","trafilatura","readability_lxml"] if kind=="light" else [sys.argv[3]]
f=["webbed_xpath_template","lxml_xpath","selectolax_css","bs4_css","casebible_regex_template_v1","repo_python_port","chat_history_manager"] if kind=="light" else []
json.dump([["generic",t,py] for t in g]+[["fb",t,py] for t in f],open(f"spec_{kind}.json","w"))
