import { Layout, Typography, Tag, Space } from 'antd';

const { Header, Content, Footer } = Layout;
const { Title, Text } = Typography;

export default function App() {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center' }}>
        <Title level={3} style={{ color: '#fff', margin: 0 }}>
          EBC-X
        </Title>
        <Text style={{ color: '#ffffffaa', marginLeft: 16 }}>
          Enterprise Business &amp; Industrial Operating System
        </Text>
      </Header>
      <Content style={{ padding: '48px' }}>
        <Space direction="vertical" size="large">
          <Title level={2}>EV1 — Enterprise Core / Foundation Implementation</Title>
          <Space>
            <Tag color="green">EV0-SPEC v1.1 PASS</Tag>
            <Tag color="green">EV0-DESIGN v1.1 PASS</Tag>
            <Tag color="gold">EV0-TASKS v1.2 PASS</Tag>
            <Tag color="blue">EV1 AUTHORIZED</Tag>
          </Space>
          <Text type="secondary">
            Modular Monolith + Event-Native + Evidence-First + Governed Agent
          </Text>
        </Space>
      </Content>
      <Footer>
        <Text type="secondary">
          EBC-X © 2026 — HTKIS Global Delivery · 华为云 Core Engineering
        </Text>
      </Footer>
    </Layout>
  );
}