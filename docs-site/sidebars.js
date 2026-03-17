/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docsSidebar: [
    'intro',
    'getting-started',
    'quickstart',
    'concepts',
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/overview',
        'architecture/p2p-layer',
        'architecture/x402-payments',
        'architecture/crdt-marketplace',
      ],
    },
    {
      type: 'category',
      label: 'Guides',
      items: [
        'guides/register-agent',
        'guides/execute-agent',
        'guides/payment-flow',
        'guides/deploy',
      ],
    },
    {
      type: 'category',
      label: 'Smart Contracts',
      items: [
        'contracts/agent-registry',
        'contracts/reputation',
        'contracts/payment-vault',
      ],
    },
    'sdk-reference',
    'api-reference',
  ],
};

module.exports = sidebars;
