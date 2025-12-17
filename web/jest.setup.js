/// <reference types="jest" />
require("@testing-library/jest-dom");

process.env.VITE_API_URL = "http://localhost:8080";
process.env.VITE_WS_URL = "ws://localhost:8080/engine/ws";
