import { Light as SyntaxHighlighter } from "react-syntax-highlighter";

// Import common languages to keep bundle size reasonable
import json from "react-syntax-highlighter/dist/esm/languages/hljs/json";
import markdown from "react-syntax-highlighter/dist/esm/languages/hljs/markdown";
import javascript from "react-syntax-highlighter/dist/esm/languages/hljs/javascript";
import typescript from "react-syntax-highlighter/dist/esm/languages/hljs/typescript";
import go from "react-syntax-highlighter/dist/esm/languages/hljs/go";
import bash from "react-syntax-highlighter/dist/esm/languages/hljs/bash";
import yaml from "react-syntax-highlighter/dist/esm/languages/hljs/yaml";
import xml from "react-syntax-highlighter/dist/esm/languages/hljs/xml";
import css from "react-syntax-highlighter/dist/esm/languages/hljs/css";
import python from "react-syntax-highlighter/dist/esm/languages/hljs/python";
import sql from "react-syntax-highlighter/dist/esm/languages/hljs/sql";
import java from "react-syntax-highlighter/dist/esm/languages/hljs/java";
import cpp from "react-syntax-highlighter/dist/esm/languages/hljs/cpp";
import dockerfile from "react-syntax-highlighter/dist/esm/languages/hljs/dockerfile";

SyntaxHighlighter.registerLanguage("json", json);
SyntaxHighlighter.registerLanguage("markdown", markdown);
SyntaxHighlighter.registerLanguage("javascript", javascript);
SyntaxHighlighter.registerLanguage("typescript", typescript);
SyntaxHighlighter.registerLanguage("go", go);
SyntaxHighlighter.registerLanguage("bash", bash);
SyntaxHighlighter.registerLanguage("yaml", yaml);
SyntaxHighlighter.registerLanguage("xml", xml);
SyntaxHighlighter.registerLanguage("css", css);
SyntaxHighlighter.registerLanguage("python", python);
SyntaxHighlighter.registerLanguage("sql", sql);
SyntaxHighlighter.registerLanguage("java", java);
SyntaxHighlighter.registerLanguage("cpp", cpp);
SyntaxHighlighter.registerLanguage("dockerfile", dockerfile);
