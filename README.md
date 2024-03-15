[![Go Report Card](https://goreportcard.com/badge/github.com/nickzer0/GoBoxer)](https://goreportcard.com/report/github.com/nickzer0/GoBoxer)
# GoBoxer

GoBoxer is a tool I started writing to learn about Websockets, and ended up getting out of hand. It's still far from complete, but can be used for managing Red Team infrastructure and projects, and can help with the following things:

- VPS creation across several Cloud providers.
- Domain name purchasing from various providers.
- Server provisioning using Ansible to execute tasks on infrastructure.
- Domain fronting/redirection using Cloudfront in AWS.
- Project/infrastructure allocation and management.

It's far from complete and there are bugs, but it's currently in a workable state.

![Alt text](/screenshots/dashboard.png?raw=true "Dashboard")

## Contributing
Contributions to GoBoxer are welcome. Whether you're looking to fix bugs, add new features, or improve documentation, your help is appreciated.

To contribute:

1. Fork the repository.
2. Create a new branch for your feature or fix.
3. Commit your changes.
4. Push your branch and submit a pull request.

## Disclaimer
GoBoxer is intended for educational and research purposes only. The authors and contributors are not responsible for any misuse or damage caused by this software. 

## Support
For support, questions, or to discuss related topics, open an issue on the GitHub repository. Contributions and feedback are always welcome.
You can reach out to me on [Twitter](https://twitter.com/_nickzer0).

## Known Issues
Pull requests are welcome if anyone would like to have a go at fixing these.
- [ ] No backend privilege separation between projects / access controls.
- [ ] DNS updates are completely broken for now - I've started implementing AWS Route 53 to use Hosted Zones, but this has so much work that needs doing.
