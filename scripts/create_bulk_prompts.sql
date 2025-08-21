-- Create bulk-specific prompts for analyzing multiple emails
-- This script should be run against your Gmail TUI database

-- First, make sure the prompt_templates table exists
CREATE TABLE IF NOT EXISTS prompt_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    prompt_text TEXT NOT NULL,
    category TEXT DEFAULT 'general',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_favorite BOOLEAN DEFAULT 0,
    usage_count INTEGER DEFAULT 0
);

-- 1. Cloud Product Analysis (for AWS/cloud newsletters)
INSERT OR REPLACE INTO prompt_templates (
    name, 
    description, 
    prompt_text, 
    category, 
    is_favorite
) VALUES (
    'Cloud Product Analysis',
    'Analyze cloud product updates and extract relevant information about specific services',
    'You are analyzing a collection of cloud product update emails. Focus on extracting and summarizing information about cloud services, new features, and product announcements.

Emails to analyze:
{{body}}

Please provide a comprehensive analysis including:
1. **New Product Features**: List any new features or capabilities mentioned
2. **Service Updates**: Document any service improvements or changes
3. **AI/ML Services**: Highlight any updates related to AI, machine learning, or Bedrock services
4. **Pricing Changes**: Note any pricing updates or new pricing models
5. **Regional Availability**: Document any new region launches or availability changes
6. **Integration Updates**: List any new integrations or API changes
7. **Security & Compliance**: Note any security enhancements or compliance updates

Format your response clearly with bullet points and sections.',
    'bulk_analysis',
    1
);

-- 2. Newsletter Digest
INSERT OR REPLACE INTO prompt_templates (
    name, 
    description, 
    prompt_text, 
    category, 
    is_favorite
) VALUES (
    'Newsletter Digest',
    'Create a concise digest summarizing the key points from multiple newsletter emails',
    'You are creating a digest from multiple newsletter emails. Extract the most important information and create a concise summary.

Emails to analyze:
{{body}}

Please create a digest with:
1. **Top Headlines**: 3-5 most important stories or announcements
2. **Key Updates**: Significant changes or new information
3. **Action Items**: Any items requiring attention or follow-up
4. **Trends**: Patterns or recurring themes across the emails
5. **Summary**: 2-3 sentence executive summary

Keep the digest concise and actionable.',
    'bulk_analysis',
    1
);

-- 3. Technical Updates Summary
INSERT OR REPLACE INTO prompt_templates (
    name, 
    description, 
    prompt_text, 
    category, 
    is_favorite
) VALUES (
    'Technical Updates Summary',
    'Summarize technical updates and changes from multiple technical emails',
    'You are analyzing technical update emails to extract key technical changes and improvements.

Emails to analyze:
{{body}}

Please provide a technical summary including:
1. **API Changes**: Any new endpoints, deprecations, or breaking changes
2. **Performance Improvements**: Speed, efficiency, or scalability enhancements
3. **New Integrations**: Third-party service connections or partnerships
4. **Security Updates**: Security patches, authentication changes, or compliance updates
5. **Developer Experience**: Tools, SDKs, or development workflow improvements
6. **Infrastructure Changes**: Platform updates, deployment changes, or architecture improvements
7. **Migration Notes**: Any required actions for existing users

Format with clear technical details and impact assessment.',
    'bulk_analysis',
    1
);

-- 4. Business Intelligence Report
INSERT OR REPLACE INTO prompt_templates (
    name, 
    description, 
    prompt_text, 
    category, 
    is_favorite
) VALUES (
    'Business Intelligence Report',
    'Extract business insights and strategic information from multiple business emails',
    'You are analyzing business emails to extract strategic insights and business intelligence.

Emails to analyze:
{{body}}

Please provide a business intelligence report including:
1. **Market Trends**: Industry developments or market shifts
2. **Competitive Intelligence**: Competitor activities or positioning
3. **Strategic Initiatives**: New business directions or partnerships
4. **Financial Updates**: Revenue, investment, or cost information
5. **Customer Insights**: User feedback, adoption metrics, or satisfaction data
6. **Risk Factors**: Potential challenges or concerns
7. **Opportunities**: New market opportunities or growth areas
8. **Recommendations**: Strategic actions or next steps

Format as a business report with clear insights and actionable recommendations.',
    'bulk_analysis',
    1
);

-- 5. Event & Conference Summary
INSERT OR REPLACE INTO prompt_templates (
    name, 
    description, 
    prompt_text, 
    category, 
    is_favorite
) VALUES (
    'Event & Conference Summary',
    'Summarize information from multiple event-related emails',
    'You are analyzing event and conference emails to create a comprehensive summary.

Emails to analyze:
{{body}}

Please provide an event summary including:
1. **Upcoming Events**: Dates, locations, and key details
2. **Registration Deadlines**: Important dates and requirements
3. **Featured Speakers**: Key presenters and their topics
4. **Session Highlights**: Notable sessions, workshops, or tracks
5. **Networking Opportunities**: Meetups, social events, or community activities
6. **Costs & Discounts**: Pricing, early bird offers, or special rates
7. **Travel Information**: Venue details, accommodation, or transportation
8. **Action Items**: Registration tasks, preparation requirements, or follow-ups

Format with clear event details and next steps.',
    'bulk_analysis',
    1
);

-- 6. Amazon AI Analysis
INSERT OR REPLACE INTO prompt_templates (
    name, 
    description, 
    prompt_text, 
    category, 
    is_favorite
) VALUES (
    'Amazon AI Analysis',
    'Analyze emails related to Amazon AI tools and services (Bedrock, Nova, etc.)',
    'You are analyzing a collection of emails related to Amazon AI tools and services. Focus on extracting and summarizing information about Amazon''s AI offerings, particularly Amazon Bedrock, Amazon Nova, and other AI-related services.

Emails to analyze:
{{messages}}

Please provide a comprehensive analysis including:

## 🚀 **Amazon AI Services Overview**
- **Amazon Bedrock**: Document any updates, new models, features, or capabilities
- **Amazon Nova**: Note any mentions, updates, or new features
- **Other AI Services**: Identify any other Amazon AI tools or services mentioned

## 🔧 **Technical Updates**
- **New Models**: List any new AI models or model versions announced
- **API Changes**: Document any API updates, new endpoints, or integration changes
- **Performance Improvements**: Note any performance enhancements or optimizations
- **New Features**: Highlight any new capabilities or functionality

## 💰 **Business & Pricing**
- **Pricing Updates**: Note any pricing changes, new pricing models, or cost optimizations
- **Availability**: Document any new region launches or availability changes
- **Enterprise Features**: Identify any enterprise-focused capabilities or updates

## 🔗 **Integrations & Partnerships**
- **Third-party Integrations**: List any new integrations with external tools or services
- **AWS Service Integration**: Note how these AI services integrate with other AWS services
- **Partner Ecosystem**: Document any partner announcements or collaborations

## 📊 **Use Cases & Applications**
- **Industry Applications**: Identify specific industries or use cases mentioned
- **Customer Success Stories**: Note any customer implementations or case studies
- **Best Practices**: Document any recommended practices or guidelines

## 🔒 **Security & Compliance**
- **Security Features**: Note any security enhancements or new security capabilities
- **Compliance Updates**: Document any compliance certifications or regulatory updates
- **Privacy Features**: Identify any privacy-related capabilities or improvements

## 📈 **Market Position & Competition**
- **Competitive Advantages**: Note any features that differentiate Amazon''s AI offerings
- **Market Trends**: Identify any broader AI market trends mentioned
- **Future Roadmap**: Document any hints about upcoming features or direction

## 🎯 **Action Items & Recommendations**
- **Key Takeaways**: Summarize the 3-5 most important points
- **Follow-up Required**: Identify any items requiring attention or further research
- **Strategic Implications**: Note any business or technical implications

Format your response clearly with bullet points, sections, and actionable insights. Focus on providing value for someone managing or evaluating Amazon AI services for their organization.',
    'bulk_analysis',
    1
);

-- Verify the prompts were created
SELECT id, name, description, category FROM prompt_templates WHERE category = 'bulk_analysis';

