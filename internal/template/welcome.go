package template

import "fmt"

// DefaultIndexPage returns an HTML welcome page for a newly created domain.
func DefaultIndexPage(domainName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s — Welcome</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Sora:wght@600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --pink: #E84393;
            --pink-light: #FD79A8;
            --pink-deep: #C22D78;
            --violet: #6C5CE7;
            --dark: #0D0D12;
            --dark-card: #15151D;
            --dark-border: #1E1E2A;
            --gray: #6B6B80;
            --bg: #0A0A10;
            --white: #F8F8FC;
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Inter', sans-serif;
            background: var(--bg);
            color: var(--white);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            -webkit-font-smoothing: antialiased;
        }

        .container {
            text-align: center;
            max-width: 520px;
            padding: 40px 24px;
        }

        .icon {
            width: 80px;
            height: 80px;
            margin: 0 auto 32px;
            border-radius: 20px;
            background: linear-gradient(135deg, var(--pink), var(--violet));
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 36px;
            animation: float 6s ease-in-out infinite;
        }

        @keyframes float {
            0%%, 100%% { transform: translateY(0px); }
            50%% { transform: translateY(-10px); }
        }

        h1 {
            font-family: 'Sora', sans-serif;
            font-size: 2rem;
            font-weight: 700;
            margin-bottom: 12px;
            background: linear-gradient(135deg, var(--pink-light), var(--pink));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .domain {
            font-family: 'Inter', sans-serif;
            font-size: 1.1rem;
            color: var(--gray);
            margin-bottom: 32px;
        }

        .card {
            background: var(--dark-card);
            border: 1px solid var(--dark-border);
            border-radius: 12px;
            padding: 24px;
            margin-bottom: 24px;
        }

        .card p {
            color: var(--gray);
            line-height: 1.7;
            font-size: 0.95rem;
        }

        .badge {
            display: inline-block;
            padding: 6px 16px;
            border-radius: 100px;
            font-size: 0.8rem;
            font-weight: 500;
            background: rgba(232, 67, 147, 0.1);
            color: var(--pink-light);
            border: 1px solid rgba(232, 67, 147, 0.2);
        }

        .footer {
            margin-top: 40px;
            color: var(--gray);
            font-size: 0.8rem;
        }

        .footer a {
            color: var(--pink);
            text-decoration: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">P</div>
        <h1>Welcome!</h1>
        <p class="domain">%s</p>
        <div class="card">
            <p>
                Your domain has been successfully configured and is ready to go.
                Upload your website files to the document root to get started.
            </p>
        </div>
        <span class="badge">Powered by PinkPanel</span>
        <div class="footer">
            <p>Managed with <a href="https://github.com/furkankufrevi/PinkPanel">PinkPanel</a></p>
        </div>
    </div>
</body>
</html>`, domainName, domainName)
}
