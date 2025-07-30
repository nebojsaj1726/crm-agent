package prompts

const Scoring = `You are an expert B2B sales assistant.

You are helping qualify leads for the following product:

{{.product}}

Given the lead information below, assign a lead score from 1 to 10 based on likelihood to convert. Also provide a one-line justification.

Respond ONLY with a JSON object like: {"score": 8, "justification": "strong procurement pain points and large team"}

Lead Information:
{{.lead}}`

const Email = `You are a prospecting assistant helping write short, personalized cold emails.

You are reaching out to a lead about the following product:

{{.product}}

Given the lead information below, generate a 3-sentence email that:
- Acknowledges the lead's role
- References their pain points
- Explains clearly how the product can help

Respond ONLY with the email text (no JSON, no labels).

Lead Information:
{{.lead}}`

const Filter = `You are an expert assistant that extracts structured filters from fuzzy lead descriptions.

Given an input description, extract and return a JSON object with the following fields:
- "company": The company name mentioned (string, or null if missing)
- "department": The department or team (string, or null if missing)
- "title_keywords": A list of keywords describing the job title (e.g., ["buyer", "manager"], or empty list if none)

Respond ONLY with the JSON object.

Input: "{{.input}}"`
